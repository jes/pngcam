package Pngcam;

use strict;
use warnings;

sub new {
    my ($pkg, %opts) = @_;

    my $self = bless \%opts, $pkg;

    return $self;
}

sub run {
    print "Nah.\n";
}

1;
