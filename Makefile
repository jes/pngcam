build/pngcam: build.header lib/Pngcam.pm pngcam
	mkdir -p build/
	cat $^ | sed 's/^use Pngcam/import Pngcam/' > build/pngcam
	chmod +x build/pngcam

install: build/pngcam
	install -m 0755 build/pngcam /usr/bin/pngcam

clean:
	rm -f build/pngcam
	rm -f t/data/*.new

test:
	prove -l t/
