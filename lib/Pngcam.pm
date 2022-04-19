package Pngcam;

use strict;
use warnings;

use GD;
use List::Util qw(min max);
use POSIX qw(floor);

sub new {
    my ($pkg, %opts) = @_;

    my $self = bless \%opts, $pkg;

    $self->{image} = GD::Image->newFromPng($self->{image_file}, 1);
    ($self->{pxwidth}, $self->{pxheight}) = $self->{image}->getBounds;

    my $aspect_ratio = $self->{pxwidth} / $self->{pxheight};
    $self->{width} //= $self->{height} * $aspect_ratio;
    $self->{height} //= $self->{width} / $aspect_ratio;
    $self->{step_forward} //= $self->{step_over};
    $self->{step_forward} = $self->{step_over} if $self->{step_forward} <= 0;

    $self->{x_px_mm} = $self->{pxwidth} / $self->{width};
    $self->{y_px_mm} = $self->{pxheight} / $self->{height};

    $self->{max_colour} = $self->{rgb} ? 16777215 : 255;

    if ($self->{write_stock}) {
        $self->{write_stock_image} = GD::Image->new($self->{pxwidth}, $self->{pxheight}, 1);

        # clear to white
        my $white = $self->{image}->colorAllocate(255,255,255);
        $self->{write_stock_image}->filledRectangle(0, 0, $self->{pxwidth}, $self->{pxheight}, $white);
    }

    return $self;
}

sub run {
    my ($self) = @_;

    if (!$self->{quiet}) {
        my $unit = $self->{imperial} ? 'inches' : 'mm';
        print STDERR "$self->{pxwidth}x$self->{pxheight} px depth map. $self->{width}x$self->{height} $unit work piece.\n";
        print STDERR "X resolution is $self->{x_px_mm} px/$unit. Y resolution is $self->{y_px_mm} px/$unit.\n";
        print STDERR "Step-over is $self->{step_over} $unit = " . sprintf("%.2f", $self->{step_over} * $self->{x_px_mm}) . " px in X and " . sprintf("%.2f", $self->{step_over} * $self->{y_px_mm}) . " px in Y\n";
        print STDERR "Step-forward is $self->{step_forward} $unit = " . sprintf("%.2f", $self->{step_forward} * $self->{x_px_mm}) . " px in X and " . sprintf("%.2f", $self->{step_forward} * $self->{y_px_mm}) . " px in Y\n";
        print STDERR "\n";
    }

    if ($self->{normalise}) {
        $self->scan_brightness;
    }

    # setup defaults
    print $self->{imperial} ? "G20\n" : "G21\n"; # units in inches or mm
    print "G90\n"; # absolute coordinates
    print "G54\n"; # work coordinate system

    # start spindle
    print "M3 S$self->{spindle_speed}\n";

    if ($self->{route} =~ /^(both|horizontal)$/) {
        $self->one_pass('h');
    }

    # XXX: "unlimited" stepdown on second pass when doing 2 passes, as the material
    # was already roughed out by the first pass
    $self->{step_down} = 10000 if $self->{route} eq 'both';

    if ($self->{route} =~ /^(both|vertical)$/) {
        $self->one_pass('v');
    }

    # stop spindle
    print "M5\n";

    # save stock heightmap
    $self->{write_stock_image}->_file($self->{write_stock}) if $self->{write_stock};
}

# direction = 'h' or 'v'
sub one_pass {
    my ($self, $direction) = @_;

    print "(Start $direction pass)\n";

    # move to origin
    print "G91 G1 Z$self->{rapid_clearance} F$self->{rapid_feedrate}\n"; # raise up 5mm relative to whatever Z is currently at
    print "G90 G0 X$self->{x_offset} Y$self->{y_offset} F$self->{rapid_feedrate}\n"; # absolute coordinates, move to start point

    my ($x, $y, $z) = (0, 0, 0); # mm

    my $xstep = ($direction eq 'h') ? $self->{step_forward} : 0;
    my $ystep = ($direction eq 'v') ? $self->{step_forward} : 0;

    my @path;

    print STDERR "Generating path: 0%" if !$self->{quiet};

    my ($xlimit,$ylimit);
    if ($direction eq 'h') {
        $xlimit = $self->{width}+$self->{step_forward};
        $ylimit = $self->{height}+$self->{step_over};
    } else {
        $xlimit = $self->{width}+$self->{step_over};
        $ylimit = $self->{height}+$self->{step_forward};
    }

    # generate path to set Z position at each X/Y position encountered
    while ($x >= 0 && $y >= 0 && $x < $xlimit && $y < $ylimit) {
        while ($x >= 0 && $y >= 0 && $x < $xlimit && $y < $ylimit) {
            my $p = {
                x => $x,
                y => -$y, # note: negative
                z => $self->cut_depth($x, $y),
                G => 'G1',
            };
            push @path, $p;

            if ($self->{write_stock} && !$self->{roughing_only} && @path >= 2) {
                $self->plot_move($path[$#path-1], $p);
            }

            $x += $xstep; $y += $ystep;
        }

        my $pct;
        if ($direction eq 'h') {
            $pct = sprintf("%2d", 100 * $y / $self->{height});
            $xstep = -$xstep;
            $x += $xstep;
            $y += $self->{step_over};
        } else {
            $pct = sprintf("%2d", 100 * $x / $self->{width});
            $ystep = -$ystep;
            $y += $ystep;
            $x += $self->{step_over};
        }
        print STDERR "   \rGenerating path: $pct%" if !$self->{quiet};
    }

    if (!$self->{quiet}) {
        print STDERR "\rGenerating path: done.";
        print STDERR "\nPost-processing...";
    }

    # postprocess path to limit maximum stepdown
    my @extrapath;
    my $last = { x=>0, y=>0, z=>0 };
    my $deepest = -$self->{depth} - ($self->{deep_black} ? ($self->{tool_diameter}/2) : 0);
    for (my $zheight = -$self->{step_down}; $zheight > $deepest; $zheight -= $self->{step_down}) {
        my $cutting = 0;
        for my $p (@path) {
            if ($p->{z} < $zheight) {
                # if we're not already cutting into the work, and the new point isn't adjacent to the last one, move the tool up and over
                if (!$cutting && !$self->adjacent_points($last->{x}, $last->{y}, $p->{x}, $p->{y})) {
                    push @extrapath, {
                        x => $last->{x},
                        y => $last->{y},
                        z => $self->{rapid_clearance},
                        G => 'G1',
                    };
                    push @extrapath, {
                        x => $p->{x},
                        y => $p->{y},
                        z => $self->{rapid_clearance},
                        G => 'G0',
                    };
                    if ($zheight + $self->{step_down} + $self->{rapid_clearance} < $self->{rapid_clearance}) {
                        # rapidly move down to $rapid_clearance above where the last cut depth was
                        push @extrapath, {
                            x => $p->{x},
                            y => $p->{y},
                            z => $zheight + $self->{step_down} + $self->{rapid_clearance},
                            G => '_G0', # XXX: this will get turned into a G1 but allowed $rapid_feedrate
                        };
                    }
                }
                push @extrapath, {
                    x => $p->{x},
                    y => $p->{y},
                    z => $zheight,
                    G => 'G1',
                };
                if ($self->{write_stock} && $self->{roughing_only} && @extrapath >= 2 && ($p->{z} > $zheight-$self->{step_down} || $extrapath[$#extrapath-1]->{z} > $zheight-$self->{step_down})) {
                    $self->plot_move($extrapath[$#extrapath-1], $extrapath[$#extrapath]);
                }
                $last = $extrapath[$#extrapath];
                $cutting = 1;
            } else {
                $cutting = 0;
            }
        }
    }
    if (@extrapath) {
        # if we did a roughing step, then move back to origin before starting the real cuts
        push @extrapath, {
            x => $last->{x},
            y => $last->{y},
            z => $self->{rapid_clearance},
            G => 'G1',
        };
        push @extrapath, {
            x => 0,
            y => 0,
            z => $self->{rapid_clearance},
            G => 'G0',
        };
    }
    if ($self->{roughing_only}) {
        @path = @extrapath;
    } else {
        @path = (@extrapath, @path);
    }

    # postprocess path to combine straight lines into a single larger run
    my $i = 2;
    while ($i < @path) {
        my $first = $path[$i-2];
        my $prev = $path[$i-1];
        my $cur = $path[$i];

        # don't combine path segments of different speed
        if ($prev->{G} ne $cur->{G}) {
            $i++;
            next;
        }

        my $prev_xy = atan2($prev->{y}-$first->{y}, $prev->{x}-$first->{x});
        my $cur_xy = atan2($cur->{y}-$prev->{y}, $cur->{x}-$prev->{x});
        my $prev_xz = atan2($prev->{z}-$first->{z}, $prev->{x}-$first->{x});
        my $cur_xz = atan2($cur->{z}-$prev->{z}, $cur->{x}-$prev->{x});
        my $prev_yz = atan2($prev->{z}-$first->{z}, $prev->{y}-$first->{y});
        my $cur_yz = atan2($cur->{z}-$prev->{z}, $cur->{y}-$prev->{y});

        my $epsilon = 0.0001; # consider 2 angles equal if they are within this error

        # if the route first->prev has the same angle as prev->cur, then first->prev->cur is a straight line,
        # so we can remove prev and just go straight from first->cur

        if (abs($cur_xy - $prev_xy) < $epsilon && abs($cur_xz - $prev_xz) < $epsilon && abs($cur_yz - $prev_yz) < $epsilon) {
            # delete prev (the element at index $i-1) from the path
            splice @path, $i-1, 1;
        } else {
            # move onto next path segment
            $i++;
        }
    }

    print STDERR "\nWriting output..." if !$self->{quiet};

    $last = {
        x => 0,
        y => 0,
        z => 0,
    };
    my $cycletime = 0;
    for my $p (@path) {
        # calculate the maximum feed rate that will not cause movement in either the XY plane or the Z axis to exceed their configured feed rates
        my $dx = $p->{x} - $last->{x};
        my $dy = $p->{y} - $last->{y};
        my $xy_dist = sqrt($dx*$dx + $dy*$dy);
        my $z_dist = $p->{z} - $last->{z};
        my $total_dist = sqrt($xy_dist*$xy_dist + $z_dist*$z_dist);

        my $feed_rate;
        if ($p->{G} eq '_G0') {
            # XXX: turn _G0 into a fast G1 (this is used for quickly lowering the tool down to where it needs to start
            # cutting during the roughing phase, but not used for actual cuts)
            $p->{G} = 'G1';
            $feed_rate = $self->{rapid_feedrate};
        } elsif ($p->{G} eq 'G0' || ($xy_dist == 0 && $z_dist > 0)) {
            $feed_rate = $self->{rapid_feedrate};
        } elsif ($p->{G} eq 'G1') {
            if ($z_dist >= 0 || (abs($xy_dist/$z_dist) > abs($self->{xy_feedrate}/$self->{z_feedrate}))) {
                # XY motion is limiting factor on speed (moving either flat or upwards in z)
                # we could do this:
                #    $feed_rate = ($total_dist / $xy_dist) * $self->{xy_feedrate};
                # but seems safer to limit the total feed rate to the configured XY feed rate, maybe revisit this:
                $feed_rate = $self->{xy_feedrate};
            } else {
                # Z motion is limiting factor on speed
                $feed_rate = abs($total_dist / $z_dist) * $self->{z_feedrate};
            }

            $feed_rate = $self->{xy_feedrate} if $feed_rate > $self->{xy_feedrate}; # XXX: can this happen?
        }

        $cycletime += $self->movetime($last->{x},$last->{y},$last->{z}, $p->{x},$p->{y},$p->{z}, $feed_rate);

        print sprintf("$p->{G} X%.4f Y%.4f Z%.4f F%.1f\n", $p->{x}+$self->{x_offset}, $p->{y}+$self->{y_offset}, $p->{z}+$self->{z_offset}, $feed_rate);
        $last = $p;
    }

    # pick up the tool at the end of the path
    print "G1 Z$self->{rapid_clearance} F$self->{rapid_feedrate}\n";

    print STDERR "\nDone.\n" if !$self->{quiet};

    print STDERR "Cycle time estimate: $cycletime secs\n" if !$self->{quiet};
}

# return 1 if the 2 points are orthogonal and 1 stepover apart
sub adjacent_points {
    my ($self, $x1, $y1, $x2, $y2) = @_;

    my $epsilon = 0.0001;

    if (abs($x2 - $x1) < $epsilon) {
        return (abs($y2 - $y1) - $self->{step_over}) < $epsilon;
    } elsif (abs($y2 - $y1) < $epsilon) {
        return (abs($x2 - $x1) - $self->{step_over}) < $epsilon;
    }
}

# return the required depth centred at (x,y) mm, taking into account the tool size and shape and work clearance
sub cut_depth {
    my ($self, $x, $y) = @_;

    my $tool_radius = $self->{tool_diameter}/2 + $self->{clearance};

    my $black_probe_depth = -$self->{depth} - $tool_radius + $self->{clearance};

    my @depths = ($black_probe_depth - 1);

    # attempt to sample every pixel in the circular footprint under the tool
    # TODO: this can get pretty slow, perhaps we should instead sample a fixed number of pixels in a
    # sensible pattern? Also, this seems ripe for a dynamic programming solution, but I can't quite see one
    for (my $sy = -$tool_radius; $sy <= $tool_radius; $sy += (1 / $self->{y_px_mm})) {
        for (my $sx = -$tool_radius; $sx <= $tool_radius; $sx += (1 / $self->{x_px_mm})) {
            my $rx = sqrt($sx*$sx + $sy*$sy); # rx is radius from centre of ball in x/y plane
            next if $rx > $tool_radius;

            my $zoffset = $self->{clearance};
            if ($self->{tool_shape} eq 'ball') {
                # use Pythagoras to calculate z height at radius $rx in x/y plane from centre of ball
                $zoffset = sqrt($tool_radius*$tool_radius - $rx*$rx) - $tool_radius + $self->{clearance};
            }

            # only add this depth sample if we're not in deep-black mode, or if this point isn't black
            if (!$self->{deep_black} || !$self->is_black($x+$sx, $y+$sy)) {
                push @depths, $self->get_depth($x+$sx, $y+$sy)+$zoffset;
            }
        }
    }

    my $max_depth = max(@depths);

    # if the cut is all on black colour then just cut to $self->{depth} instead of below
    if ($max_depth < $black_probe_depth) {
        $max_depth = -$self->{depth} + $self->{clearance};
    }

    return $max_depth;
}

# return 1 if the pixel at (x,y) mm is black, 0 otherwise
sub is_black {
    my ($self, $x, $y) = @_;

    my $brightness = $self->get_brightness($x * $self->{x_px_mm}, $y * $self->{y_px_mm});

    return ($brightness == 0);
}

# return the depth at (x,y) mm
sub get_depth {
    my ($self, $x, $y) = @_;

    my $brightness = $self->get_brightness($x * $self->{x_px_mm}, $y * $self->{y_px_mm});

    # brightness=0 is the bottom of the cut, so max. negative Z
    return ($brightness - $self->{max_colour}) * ($self->{depth} / $self->{max_colour});
}

# scan the heightmap and fill in $self->{min_bright} and $self->{max_bright}
sub scan_brightness {
    my ($self) = @_;

    my $minbright = 16777216;
    my $maxbright = -1;

    for my $y (0 .. $self->{pxheight}-1) {
        for my $x (0 .. $self->{pxwidth}-1) {
            my $col = $self->{image}->getPixel($x, $y);
            my ($r,$g,$b) = $self->{image}->rgb($col);
            my $brightness = $self->{rgb} ? 65536*$r+256*$g+$b : ($r+$g+$b)/3;
            $minbright = $brightness if $brightness < $minbright;
            $maxbright = $brightness if $brightness > $maxbright;
        }
    }

    $self->{min_bright} = $minbright;
    $self->{max_bright} = $maxbright;
}

# return pixel brightness at (x,y) pixels, 0..255 (or 0..16777215)
# this also applies normalisation and inversion
sub get_brightness {
    my ($self, $x, $y) = @_;

    $x = floor($x);
    $y = floor($y);

    $x = $self->{pxwidth}-1-$x if $self->{x_flip};
    $y = $self->{pxheight}-1-$y if $self->{y_flip};

    if ($x < 0 || $y < 0 || $x >= $self->{pxwidth} || $y >= $self->{pxheight}) {
        return 0;
    }

    # TODO: interpolate pixels at:
    # floor(x),floor(y)
    # floor(x),ceil(y)
    # ceil(x),floor(y)
    # ceil(x),ceil(y)

    my $col = $self->{image}->getPixel($x, $y);
    my ($r,$g,$b) = $self->{image}->rgb($col);
    my $brightness = $self->{rgb} ? $r*65536+$g*256+$b : ($r+$g+$b)/3;

    if ($self->{normalise}) {
        if (!$self->{normalise_ignore_black} || $brightness != 0) {
            $brightness = ($brightness - $self->{min_bright}) * ($self->{max_colour} / ($self->{max_bright} - $self->{min_bright}));
        }
    }

    if ($self->{invert}) {
        return $self->{max_colour}-$brightness;
    } else {
        return $brightness;
    }
}

# calculate time to move from ($x1,$y1,$z1) to ($x2,$y2,$z2) at $feedrate mm/min, in seconds
sub movetime {
    my ($self, $x1,$y1,$z1, $x2,$y2,$z2, $feedrate) = @_;

    # TODO: take into account max. acceleration

    $feedrate = $self->{maxvel} if !$feedrate || $feedrate > $self->{maxvel};

    my $dx = abs($x2-$x1);
    my $dy = abs($y2-$y1);
    my $dz = abs($z2-$z1);
    my $dist = sqrt($dx*$dx + $dy*$dy + $dz*$dz);
    my $mins = $dist / $feedrate;
    my $secs = $mins * 60;
    return $secs;
}

# plot a single move
sub plot_move {
    my ($self, $p1, $p2) = @_;

    my $img = $self->{write_stock_image};

    my $dx = $p2->{x}-$p1->{x};
    my $dy = $p2->{y}-$p1->{y};
    my $dz = $p2->{z}-$p1->{z};

    my $lenxyz = sqrt($dx*$dx + $dy*$dy + $dz*$dz);

    if ($lenxyz == 0) {
        $self->plot_toolpoint($img, $p1);
        return;
    }

    $dx /= $lenxyz;
    $dy /= $lenxyz;
    $dz /= $lenxyz;

    my $step = min(1/$self->{x_px_mm}, 1/$self->{y_px_mm});

    # plot interpolated path
    for (my $p = 0; $p <= $lenxyz; $p += $step) {
        my $px = $p1->{x} + $dx*$p;
        my $py = $p1->{y} + $dy*$p;
        my $pz = $p1->{z} + $dz*$p;

        $self->plot_toolpoint($img, {x=>$px, y=>$py, z=>$pz});
    }
}


# plot a single point on the toolpath
sub plot_toolpoint {
    my ($self, $img, $p, %opts) = @_;

    my $tool_radius = $self->{tool_diameter} / 2;

    my $x = $p->{x};
    my $y = $p->{y};
    my $z = $p->{z};

    # plot the depth for every pixel within the tool radius of ($x,$y);
    for (my $sy = -$tool_radius; $sy <= $tool_radius; $sy += (1 / $self->{y_px_mm})) {
        for (my $sx = -$tool_radius; $sx <= $tool_radius; $sx += (1 / $self->{x_px_mm})) {
            my $rx = sqrt($sx*$sx + $sy*$sy); # rx is radius from centre of ball in x/y plane
            next if $rx > $tool_radius;

            my $zoffset = 0;
            if ($self->{tool_shape} eq 'ball') {
                # ball end mill: spherical shape
                $zoffset = $tool_radius - sqrt($tool_radius*$tool_radius - $rx*$rx);
            }

            $self->plot_pixel($img, $x+$sx, $y+$sy, $z+$zoffset);
        }
    }
}

# plot a single pixel in the heightmap
sub plot_pixel {
    my ($self, $img, $x, $y, $z) = @_;

    return if $z > 0; # non-cut moves do nothing to the stock
    $z = -$self->{depth} if $z < -$self->{depth}; # heightmap can't represent deeper than the bottom

    # this is the inverse of $self->get_depth();
    $x = int($x * $self->{x_px_mm} + 0.5);
    $y = int(-$y * $self->{y_px_mm} + 0.5);
    my $brightness = int(($z * $self->{max_colour}) / $self->{depth} + $self->{max_colour});

    my $cur_col = $img->getPixel($x,$y);
    my ($r,$g,$b) = $img->rgb($cur_col);
    my $h = $self->{rgb} ? 65536*$r+256*$g+$b : $r;
    return if $h < $brightness; # do nothing if existing brightness is darker than what we're going to draw

    my $col;
    if ($self->{rgb}) {
        my $r = ($brightness/65536)%256;
        my $g = ($brightness/256)%256;
        my $b = $brightness%256;
        $col = $img->colorAllocate($r, $g, $b);
    } else {
        $col = $img->colorAllocate($brightness, $brightness, $brightness);
    }

    $img->setPixel($x, $y, $col);
}

1;
