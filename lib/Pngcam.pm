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

    # setup defaults
    print "G21\n"; # units in mm
    print "G90\n"; # absolute coordinates
    print "G54\n"; # work coordinate system

    # start spindle
    print "M3 S$self->{spindle_speed}\n";

    if ($self->{route} =~ /^(both|horizontal)$/) {
        $self->one_pass('h');
    }

    if ($self->{route} =~ /^(both|vertical)$/) {
        $self->one_pass('v');
    }

    # lift tool up
    print "G1 Z5 F$self->{z_feedrate}\n";

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
    print "G1 Z0 F$self->{z_feedrate}\n"; # move down to z=0

    my ($x, $y, $z) = (0, 0, 0); # mm

    my $xstep = ($direction eq 'h') ? $self->{step_over} : 0;
    my $ystep = ($direction eq 'v') ? $self->{step_over} : 0;

    my @path;

    if ($direction eq 'h') {
        while ($x >= 0 && $y >= 0 && $x <= $self->{width} && $y <= $self->{height}) {
            while ($x >= 0 && $y >= 0 && $x <= $self->{width} && $y <= $self->{height}) {
                push @path, {
                    x => $x,
                    y => -$y, # note: negative
                    z => $self->cut_depth($x, $y),
                };
                $x += $xstep; $y += $ystep;
            }
            if ($direction eq 'h') {
                $xstep = -$xstep;
                $x += $xstep;
                $y += $self->{step_over};
            } else {
                $ystep = -$ystep;
                $y += $ystep;
                $x += $self->{step_over};
            }
        }
    }

    # TODO: postprocess path to limit maximum stepdown (if route eq 'both', then limit on 1st pass only)

    # postprocess path to combine straight lines into a single larger run
    my $i = 2;
    while ($i < @path) {
        my $first = $path[$i-2];
        my $prev = $path[$i-1];
        my $cur = $path[$i];

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

    # TODO: limit Z feed rate
    printf sprintf("G1 X%.4f Y%.4f Z%.4f F%.1f\n", $_->{x}, $_->{y}, $_->{z}, $self->{xy_feedrate}) for @path;
}

# return the required depth centred at (x,y) mm, taking into account the tool size and shape and work clearance
sub cut_depth {
    my ($self, $x, $y) = @_;

    my $tool_radius = $self->{tool_diameter}/2 + $self->{clearance};

    # XXX: currently we just look at the centre and 4 perimeter depths, should instead consider every pixel
    # in the area of a circle of radius $tool_radius and calculate $zoffset with a function

    # defaults for ball-nose end mill:
    my $zoffset_centre = $self->{clearance};
    my $zoffset_perimeter = -$tool_radius;
    # override for flat end mill:
    if ($self->{tool_shape} eq 'flat') {
        $zoffset_perimeter = $self->{clearance};
    }

    # TODO: ignore samples where the colour is magenta
    my @depths = (
        $self->get_depth($x,$y)+$zoffset_centre,
        $self->get_depth($x+$tool_radius,$y)+$zoffset_perimeter,
        $self->get_depth($x,$y+$tool_radius)+$zoffset_perimeter,
        $self->get_depth($x-$tool_radius,$y)+$zoffset_perimeter,
        $self->get_depth($x,$y-$tool_radius)+$zoffset_perimeter,
    );

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
