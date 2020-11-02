#!/usr/bin/perl

use strict;
use warnings;

use File::Compare;
use Test::More;
use Pngcam;

plan tests => 5;

# test CLI

system("PERL5LIB=lib ./pngcam --quiet --rapid-clearance 2 --step-over 2.4 --width 60 --tool-shape ball --tool-diameter 4 --depth 5 --xy-feed-rate 500 --z-feed-rate 200 --step-down 2 --clearance 0.1 --route both t/data/klingon-dagger.png > t/data/dagger-top-rough.gcode.new 2> t/data/stderr.new");
check("dagger-top-rough");

system("PERL5LIB=lib ./pngcam --quiet --rapid-clearance 2 --step-over 0.6 --width 60 --tool-shape ball --tool-diameter 2 --depth 5 --xy-feed-rate 1000 --z-feed-rate 400 --route both t/data/klingon-dagger.png > t/data/dagger-top-finish.gcode.new 2>> t/data/stderr.new");
check("dagger-top-finish");

system("PERL5LIB=lib ./pngcam --quiet --rapid-clearance 2 --step-over 2.4 --width 60 --tool-shape ball --tool-diameter 4 --depth 5 --xy-feed-rate 500 --z-feed-rate 200 --step-down 2 --clearance 0.1 --route both --x-flip t/data/klingon-dagger.png > t/data/dagger-bottom-rough.gcode.new 2>> t/data/stderr.new");
check("dagger-bottom-rough");

system("PERL5LIB=lib ./pngcam --quiet --rapid-clearance 2 --step-over 0.6 --width 60 --tool-shape ball --tool-diameter 2 --depth 5 --xy-feed-rate 1000 --z-feed-rate 400 --route both --x-flip t/data/klingon-dagger.png > t/data/dagger-bottom-finish.gcode.new 2>> t/data/stderr.new");
check("dagger-bottom-finish");

is(!compare("t/data/stderr.new", "/dev/null"), 1, "CLI invocation: t/data/stderr.new == /dev/null");

done_testing;

sub check {
    my ($name) = @_;
    is(!compare("t/data/$name.gcode", "t/data/$name.gcode.new"), 1, "CLI invocation: t/data/$name.gcode.new == t/data/$name.gcode");
}
