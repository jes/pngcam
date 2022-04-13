all: build/pngcam build/pngcam-render

build/pngcam: build.header lib/Pngcam.pm pngcam
	mkdir -p build/
	cat $^ | sed 's/^use Pngcam/import Pngcam/' > build/pngcam
	chmod +x build/pngcam

build/pngcam-render: build.header lib/Pngcam/Render.pm lib/CAD/Format/STL.pm lib/CAD/Format/STL/part.pm pngcam-render
	mkdir -p build/
	cat $^ | sed 's/^use Pngcam/import Pngcam/' | sed 's/^use CAD::Format::STL/import CAD::Format::STL/' > build/pngcam-render
	chmod +x build/pngcam-render

install: build/pngcam build/pngcam-render
	install -m 0755 build/pngcam /usr/bin/pngcam
	install -m 0755 build/pngcam-render /usr/bin/pngcam-render

clean:
	rm -f build/pngcam build/pngcam-render t/data/*.new

test:
	prove -l t/
