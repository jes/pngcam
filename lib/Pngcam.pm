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

    # stop spindle
    print "M5\n";
}

# direction = 'h' or 'v'
sub one_pass {
    my ($self, $direction) = @_;

    print "(Start $direction pass)\n";

    # move to origin
    print "G91 G1 Z5\n"; # raise up 5mm relative to whatever Z is currently at
    print "G90 G0 X0 Y0\n"; # move to (0, 0) in x/y plane
    print "G1 Z0\n"; # move down to z=0

    my ($x, $y, $z) = (0, 0, 0); # mm

    # TODO: support vertical pass
    die "vertical not supported" if $direction eq 'v';

    if ($direction eq 'h') {
        # TODO: cut in left and right direction instead of only right to save time
        while ($y <= $self->{height}) {
            while ($x <= $self->{width}) {
                $z = $self->cut_depth($x, $y);
                # TODO: limit Z feed rate
                # TODO: support maximum stepdown
                print "G1 X$x Z$z F$self->{xy_feedrate}\n";
                $x += $self->{step_over};
            }
            $y += $self->{step_over};
            $x = 0;
            print "G1 Z5\n";
            print "G0 X0 Y-$y\n"; # note Y is negative
            print "G1 Z0\n";
        }
    }
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

    my $col = $self->{image}->getPixel($x, $y);
    my ($r,$g,$b) = $self->{image}->rgb($col);

    return ($r+$g+$b)/3;
}

1;
