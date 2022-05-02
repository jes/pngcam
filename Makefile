all: build/pngcam build/pngcam-render build/pngcam-plotter build/pngcam-go

build/pngcam: build.header lib/Pngcam.pm pngcam
	mkdir -p build/
	cat $^ | sed 's/^use Pngcam/import Pngcam/' > build/pngcam
	chmod +x build/pngcam

build/pngcam-render: build.header lib/Pngcam/Render.pm lib/CAD/Format/STL.pm lib/CAD/Format/STL/part.pm pngcam-render
	mkdir -p build/
	cat $^ | sed 's/^use Pngcam/import Pngcam/' | sed 's/^use CAD::Format::STL/import CAD::Format::STL/' > build/pngcam-render
	chmod +x build/pngcam-render

build/pngcam-plotter: plotter.c
	cc -o pngcam-plotter plotter.c -Wall -lm
	cp pngcam-plotter build/pngcam-plotter

build/pngcam-go: pngcam-go/*.go
	cd pngcam-go && go build
	cp pngcam-go/pngcam-go build/pngcam-go

install: build/pngcam build/pngcam-render
	install -m 0755 build/pngcam /usr/bin/pngcam
	install -m 0755 build/pngcam-render /usr/bin/pngcam-render
	install -m 0755 build/pngcam-plotter /usr/bin/pngcam-plotter
	install -m 0755 build/pngcam-go /usr/bin/pngcam-go

clean:
	rm -f build/pngcam build/pngcam-render build/pngcam-plotter pngcam-plotter t/data/*.new

test:
	prove -l t/
