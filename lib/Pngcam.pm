package Pngcam;

use strict;
use warnings;

use GD;
use List::Util qw(min max);

sub new {
    my ($pkg, %opts) = @_;

    my $self = bless \%opts, $pkg;

    $self->{image} = GD::Image->new($self->{image_file});
    ($self->{pxwidth}, $self->{pxheight}) = $self->{image}->getBounds;

    my $aspect_ratio = $self->{pxwidth} / $self->{pxheight};
    if (!defined $self->{width}) {
        $self->{width} = $self->{height} * $aspect_ratio;
    }
    if (!defined $self->{height}) {
        $self->{height} = $self->{width} / $aspect_ratio;
    }

    $self->{x_px_mm} = $self->{pxwidth} / $self->{width};
    $self->{y_px_mm} = $self->{pxheight} / $self->{height};

    return $self;
}

sub run {
    my ($self) = @_;

    print STDERR "$self->{pxwidth}x$self->{pxheight} px depth map. $self->{width}x$self->{height} mm work piece.\n";
    print STDERR "X resolution is $self->{x_px_mm} px/mm. Y resolution is $self->{y_px_mm} px/mm.\n";
    print STDERR "Step-over is $self->{step_over} mm = " . sprintf("%.2f", $self->{step_over} * $self->{x_px_mm}) . " px in X and " . sprintf("%.2f", $self->{step_over} * $self->{y_px_mm}) . " px in Y\n";
    print STDERR "\n";

    # setup defaults
    print "G21\n"; # units in mm
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
}

# direction = 'h' or 'v'
sub one_pass {
    my ($self, $direction) = @_;

    print "(Start $direction pass)\n";

    # move to origin
    print "G91 G1 Z5 F$self->{z_feedrate}\n"; # raise up 5mm relative to whatever Z is currently at
    print "G90 G0 X0 Y0 F$self->{rapid_feedrate}\n"; # move to (0, 0) in x/y plane

    my ($x, $y, $z) = (0, 0, 0); # mm

    my $xstep = ($direction eq 'h') ? $self->{step_over} : 0;
    my $ystep = ($direction eq 'v') ? $self->{step_over} : 0;

    my @path;

    # generate path to set Z position at each X/Y position encountered
    while ($x >= 0 && $y >= 0 && $x <= $self->{width} && $y <= $self->{height}) {
        while ($x >= 0 && $y >= 0 && $x <= $self->{width} && $y <= $self->{height}) {
            push @path, {
                x => $x,
                y => -$y, # note: negative
                z => $self->cut_depth($x, $y),
                G => 'G1',
            };
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
        print STDERR "   \rGenerating path: $pct%";
    }

    print STDERR "\rGenerating path: done.";
    print STDERR "\nPost-processing...";

    # postprocess path to limit maximum stepdown
    my @extrapath;
    my $last = {
        x => 0,
        y => 0,
        z => 0,
    };
    for (my $zheight = -$self->{step_down}; $zheight > -$self->{depth}; $zheight -= $self->{step_down}) {
        my $cutting = 0;
        for my $p (@path) {
            if ($p->{z} < $zheight) {
                # if we're not already cutting into the work, move the tool up and over
                if (!$cutting) {
                    push @extrapath, {
                        x => $last->{x},
                        y => $last->{y},
                        z => 5,
                        G => 'G1',
                    };
                    push @extrapath, {
                        x => $p->{x},
                        y => $p->{y},
                        z => 5,
                        G => 'G0',
                    };
                }
                # add this location to the roughing path
                push @extrapath, {
                    x => $p->{x},
                    y => $p->{y},
                    z => $zheight,
                    G => 'G1',
                };
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
            z => 5,
            G => 'G1',
        };
        push @extrapath, {
            x => 0,
            y => 0,
            z => 5,
            G => 'G0',
        };
    }
    @path = (@extrapath, @path);

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

        my $prev_xz = gradient2d($first->{x}, $first->{z}, $prev->{x}, $prev->{z});
        my $cur_xz = gradient2d($prev->{x}, $prev->{z}, $cur->{x}, $cur->{z});
        my $prev_yz = gradient2d($first->{y}, $first->{z}, $prev->{y}, $prev->{z});
        my $cur_yz = gradient2d($prev->{y}, $prev->{z}, $cur->{y}, $cur->{z});

        my $epsilon = 0.0001; # consider 2 gradients equal if they are within this error

        # if the route first->prev has the same gradient as prev->cur, then first->prev->cur is a straight line,
        # so we can remove prev and just go straight from first->cur

        if (abs($cur_xz - $prev_xz) < $epsilon && abs($cur_yz - $prev_yz) < $epsilon) {
            # delete prev (the element at index $i-1) from the path
            splice @path, $i-1, 1;
        } else {
            # move onto next path segment
            $i++;
        }
    }

    print STDERR "\nWriting output...";

    $last = {
        x => 0,
        y => 0,
        z => 0,
    };
    for my $p (@path) {
        # calculate the maximum feed rate that will not cause movement in either the XY plane or the Z axis to exceed their configured feed rates
        my $dx = $p->{x} - $last->{x};
        my $dy = $p->{y} - $last->{y};
        my $xy_dist = sqrt($dx*$dx + $dy*$dy);
        my $z_dist = abs($p->{z} - $last->{z}); # XXX: is this exactly what we want? does feeding "upwards" really need slowing down?
        my $total_dist = sqrt($xy_dist*$xy_dist + $z_dist*$z_dist);

        next if $xy_dist == 0 && $z_dist == 0;

        my $feed_rate;
        if ($z_dist == 0 || ($xy_dist/$z_dist > $self->{xy_feedrate}/$self->{z_feedrate})) {
            # XY motion is limiting factor on speed
            # we could do this:
            #    $feed_rate = ($total_dist / $xy_dist) * $self->{xy_feedrate};
            # but seems safer to limit the total feed rate to the configured XY feed rate, maybe revisit this:
            $feed_rate = $self->{xy_feedrate};
        } else {
            # Z motion is limiting factor on speed
            $feed_rate = ($total_dist / $z_dist) * $self->{z_feedrate};
        }

        $feed_rate = $self->{xy_feedrate} if $feed_rate > $self->{xy_feedrate}; # XXX: why can this happen?

        print sprintf("$p->{G} X%.4f Y%.4f Z%.4f F%.1f\n", $p->{x}, $p->{y}, $p->{z}, $feed_rate);
        $last = $p;
    }

    # pick up the tool at the end of the path
    print "G1 Z5 F$self->{z_feedrate}\n";

    print STDERR "\nDone.\n";
}

# return the required depth centred at (x,y) mm, taking into account the tool size and shape and work clearance
sub cut_depth {
    my ($self, $x, $y) = @_;

    my $tool_radius = $self->{tool_diameter}/2 + $self->{clearance};

    # TODO: ignore samples where the colour is magenta
    my @depths;

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
                $zoffset = sqrt($tool_radius*$tool_radius - $rx*$rx) + $self->{clearance};
            }
            push @depths, $self->get_depth($x+$sx, $y+$sy)+$zoffset;
        }
    }

    return max(@depths);
}

# return the depth at (x,y) mm
sub get_depth {
    my ($self, $x, $y) = @_;

    my $brightness = $self->get_brightness($x * $self->{x_px_mm}, $y * $self->{y_px_mm});

    # brightness=0 is the bottom of the cut, so max. negative Z
    return ($brightness - 255) * ($self->{depth} / 255);
}

# return pixel brightness at (x,y) pixels, 0..255
sub get_brightness {
    my ($self, $x, $y) = @_;

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

    return ($r+$g+$b)/3;
}

sub gradient2d {
    my ($x1, $y1, $x2, $y2) = @_;

    return ($x2-$x1==0) if ($y2 - $y1) == 0; # XXX: no divide by zero

    return ($x2 - $x1) / ($y2 - $y1);
}

1;
