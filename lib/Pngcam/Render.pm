package Pngcam::Render;

use strict;
use warnings;

use CAD::Format::STL;
use GD;

sub new {
    my ($pkg, %opts) = @_;

    my $self = bless \%opts, $pkg;

    $self->{triangles} = [];

    $self->{minx} = undef;
    $self->{maxx} = undef;
    $self->{miny} = undef;
    $self->{maxy} = undef;
    $self->{minz} = undef;
    $self->{maxz} = undef;

    my $stl = CAD::Format::STL->new->load($self->{stl_file});
    
    # TODO: what do we do if there is more than one part?

    for my $data ($stl->part->facets) {
        my ($normal, @verts) = @$data;
        for my $v (@verts) {
            my ($x, $y, $z) = @$v;
            $self->{minx} = $x if !defined $self->{minx} || $x < $self->{minx};
            $self->{maxx} = $x if !defined $self->{maxx} || $x > $self->{maxx};
            $self->{miny} = $y if !defined $self->{miny} || $y < $self->{miny};
            $self->{maxy} = $y if !defined $self->{maxy} || $y > $self->{maxy};
            $self->{minz} = $z if !defined $self->{minz} || $z < $self->{minz};
            $self->{maxz} = $z if !defined $self->{maxz} || $z > $self->{maxz};
        }
        push @{ $self->{triangles} }, \@verts;
    }

    $self->{mmwidth} = $self->{maxx} - $self->{minx};
    $self->{mmheight} = $self->{maxy} - $self->{miny};

    my $aspect_ratio = $self->{mmwidth} / $self->{mmheight};
    if (!defined $self->{width}) {
        $self->{width} = $self->{height} * $aspect_ratio;
    }
    if (!defined $self->{height}) {
        $self->{height} = $self->{width} / $aspect_ratio;
    }

    $self->{image} = GD::Image->new($self->{width} + $self->{border}*2, $self->{height} + $self->{border}*2, 1);

    # calculate pixels per mm
    $self->{x_px_mm} = $self->{width} / $self->{mmwidth};
    $self->{y_px_mm} = $self->{height} / $self->{mmheight};
    if ($self->{rgb}) {
        $self->{z_px_mm} = 16777215 / ($self->{maxz} - $self->{minz}); # not really "pixels" but meh
    } else {
        $self->{z_px_mm} = 255 / ($self->{maxz} - $self->{minz}); # not really "pixels" but meh
    }

    return $self;
}

sub run {
    my ($self) = @_;

    if (!$self->{quiet}) {
        my $mmz = $self->{maxz} - $self->{minz};
        my $border_x_mm = $self->{border} / $self->{x_px_mm};
        my $border_y_mm = $self->{border} / $self->{y_px_mm};
        print STDERR "$self->{width}x$self->{width} px depth map. $self->{mmwidth}x$self->{mmheight} mm work piece.\n";
        print STDERR "Work piece is $mmz mm tall in Z axis.\n";
        print STDERR "X border of $self->{border} px = $border_x_mm mm. Y border of $self->{border} px = $border_y_mm mm.\n";
        print STDERR "Output image is " . ($self->{width}+$self->{border}*2) . "x" . ($self->{height}+$self->{border}*2) . " px = " . ($self->{mmwidth}+$border_x_mm*2) . "x" . ($self->{mmheight} + $border_y_mm*2) . " mm.\n";
        print STDERR "X resolution is $self->{x_px_mm} px/mm. Y resolution is $self->{y_px_mm} px/mm.\n";
    }

    # clear to black
    my $black = $self->{image}->colorAllocate(0,0,0);
    $self->{image}->filledRectangle(0, 0, $self->{width}, $self->{height}, $black);

    # render each triangle
    # since we're drawing a heightmap, the image itself acts as the depth buffer
    my $ntriangles = @{ $self->{triangles} };
    my $done = 0;
    for my $t (@{ $self->{triangles} }) {
        my @vertices = @$t;
        $self->draw_triangle(map { $self->mm_to_px($_) } @vertices);

        $done++;
        my $pct = sprintf("%2d", 100 * $done / $ntriangles);
        print STDERR "   \rDrawing triangles: $pct%" if !$self->{quiet};
    }
    print STDERR "    \rDrawing triangles: done.\n" if !$self->{quiet};
}

sub save {
    my ($self, $file) = @_;
    return $self->{image}->_file($file);
}

sub mm_to_px {
    my ($self, $v) = @_;

    my ($x,$y,$z) = @$v;

    $y = $self->{border} + ($self->{maxy} - $y) * $self->{y_px_mm}; # note y axis is inverted

    if ($self->{bottom}) {
        # view from bottom (flip x and z)
        $x = $self->{border} + ($self->{maxx} - $x) * $self->{x_px_mm};
        $z = ($self->{maxz} - $z) * $self->{z_px_mm};
    } else {
        # view from top
        $x = $self->{border} + ($x - $self->{minx}) * $self->{x_px_mm};
        $z = ($z - $self->{minz}) * $self->{z_px_mm};
    }

    return [$x, $y, $z];
}

# give vertices in px
sub draw_triangle {
    my ($self, $v1, $v2, $v3) = @_;

    my %minx;
    my %maxx;

    my $miny;
    my $maxy;

    # 1. work out where the outline of the triangle is

    # this function will get called for every pixel that lies on the perimeter of the triangle
    my $perimeter_cb = sub {
        my ($x, $y, $z) = @_;

        # store the point at the minimum and maximum x coordinate observed on each y coordinate
        $minx{$y} = [$x, $y, $z] if !defined $minx{$y} || $x < $minx{$y}[0];
        $maxx{$y} = [$x, $y, $z] if !defined $maxx{$y} || $x > $maxx{$y}[0];

        $miny = $y if !defined $miny || $y < $miny;
        $maxy = $y if !defined $maxy || $y > $maxy;
    };

    $self->iterate_line($v1, $v2, $perimeter_cb);
    $self->iterate_line($v2, $v3, $perimeter_cb);
    $self->iterate_line($v3, $v1, $perimeter_cb);

    # 2. fill in scanlines
    for my $y ($miny .. $maxy) {
        my $startx = $minx{$y}[0];
        my $endx = $maxx{$y}[0];
        my $startz = $minx{$y}[2];
        my $endz = $maxx{$y}[2];
        my $zchange = $endz - $startz;
        my $xlength = $endx - $startx;
        for my $x ($startx .. $endx) {
            my $k = $xlength ? ($x - $startx) / $xlength : 1;
            my $z = $startz + $zchange * $k;
            $self->plot($x, $y, $z);
        }
    }
}

# plot Z at (X,Y) in the image if that is brighter than what is already there; if it is darker
# then do nothing
sub plot {
    my ($self, $x, $y, $z) = @_;

    my $col = $self->{image}->getPixel($x, $y);
    my ($r,$g,$b) = $self->{image}->rgb($col);

    my $h = $self->{rgb} ? 65536*$r+256*$g+$b : $r;

    return if $h > $z; # do nothing if existing brightness is brighter than what we're going to draw

    if ($self->{rgb}) {
        $r = int($z/65536);
        $g = int($z/256)%256;
        $b = $z%256;
    } else {
        $r = $z;
        $g = $z;
        $b = $z;
    }

    my $newcol = $self->{image}->colorAllocate($r, $g, $b);
    $self->{image}->setPixel($x, $y, $newcol);
}

# call the callback on every pixel on the line p1->p2 in 2d space (x/y); p1 and p2 are 3d
# points, and the 3rd dimension will be interpolated as well
sub iterate_line {
    my ($self, $p1, $p2, $cb) = @_;

    # visit the first point
    $cb->(map { int($_) } @$p1);

    my $dx = $p2->[0] - $p1->[0];
    my $dy = $p2->[1] - $p1->[1];
    my $dz = $p2->[2] - $p1->[2];

    my $length = sqrt($dx*$dx + $dy*$dy); # only 2d

    # if the line has 0 pixels length in 2d, only plot the 1st pixel, and avoid dividing by 0
    return if $length < 1;

    my $stepx = $dx / $length;
    my $stepy = $dy / $length;
    my $stepz = $dz / $length;

    my ($x,$y,$z) = @$p1;

    # visit each point on the line, stepping 1px at a time along a diagonal
    for (1..int($length)) {
        $x += $stepx;
        $y += $stepy;
        $z += $stepz;
        $cb->(int($x), int($y), int($z));
    }
}

1;
