#!/usr/bin/perl

use strict;
use warnings;

use Digest::SHA qw(sha256_hex);
use File::Compare;
use Test::More;
use Pngcam;

plan tests => 7;

# test CLI

system("PERL5LIB=lib ./pngcam-render --quiet --width 200 --border 0 --png t/data/keycap-top.png t/data/keycap.stl 2> t/data/stderr.new");
is(sha256file("t/data/keycap-top.png"), "685cb53edbf87aa8cd7f0dceda2f1bb9db3193d6d56291a5e587ff62ae0131cd", "correct SHA256 of keycap-top.png");

system("PERL5LIB=lib ./pngcam-render --quiet --width 200 --border 0 --bottom --png t/data/keycap-bottom.png t/data/keycap.stl 2>> t/data/stderr.new");
is(sha256file("t/data/keycap-bottom.png"), "f5c00ed12d9f0c2cd49f26f1b7aa5c65de28a17248cf7d0843363c1647d64a79", "correct SHA256 of keycap-bottom.png");

# XXX: these tests are based on what I used to make the keycap, but with some extra options
# added, in the interest of increasing test coverage; they're not a good way to make a
# keycap

system("PERL5LIB=lib ./pngcam --quiet --deep-black --rapid-clearance 1 --step-over 2 --width 27.03 --tool-shape ball --tool-diameter 4 --depth 10 --xy-feed-rate 200 --z-feed-rate 50 --step-down 1 --clearance 0.1 --route both t/data/keycap-top.png > t/data/keycap-top-rough.gcode.new 2>> t/data/stderr.new");
check("keycap-top-rough");

system("PERL5LIB=lib ./pngcam --quiet --deep-black --rapid-clearance 1 --step-over 0.6 --width 27.03 --tool-shape ball --tool-diameter 2 --depth 10 --xy-feed-rate 400 --z-feed-rate 100 --route both t/data/keycap-top.png > t/data/keycap-top-finish.gcode.new 2>> t/data/stderr.new");
check("keycap-top-finish");

system("PERL5LIB=lib ./pngcam --quiet --invert --rapid-clearance 1 --step-over 2 --width 27.03 --tool-shape ball --tool-diameter 4 --depth 10 --xy-feed-rate 200 --z-feed-rate 50 --step-down 1 --clearance 0.1 --route both t/data/keycap-bottom.png > t/data/keycap-bottom-rough.gcode.new 2>> t/data/stderr.new");
check("keycap-bottom-rough");

system("PERL5LIB=lib ./pngcam --quiet --normalise --rapid-clearance 1 --step-over 0.6 --width 27.03 --tool-shape ball --tool-diameter 2 --depth 10 --xy-feed-rate 400 --z-feed-rate 100 --route both t/data/keycap-bottom.png > t/data/keycap-bottom-finish.gcode.new 2>> t/data/stderr.new");
check("keycap-bottom-finish");

is(!compare("t/data/stderr.new", "/dev/null"), 1, "CLI invocation: t/data/stderr.new == /dev/null");

done_testing;

sub check {
    my ($name) = @_;
    is(!compare("t/data/$name.gcode", "t/data/$name.gcode.new"), 1, "CLI invocation: t/data/$name.gcode.new == t/data/$name.gcode");
}

sub sha256file {
    my ($name) = @_;

    open (my $fh, '<', $name)
        or die "can't open $name for reading: $!\n";
    my $c = join('', <$fh>);
    close $fh;

    return sha256_hex($c);
}
