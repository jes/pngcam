/* Plot heightmaps of toolpaths, for pngcam
 *
 * This program is driven by lib/Pngcam.pm
 *
 * Command line arguments are:
 *   mmwidth mmheight mmdepth pxwidth pxheight mmtooldiameter toolshape
 * For example:
 *   ./plotter 20.5 10.3 5 800 400 6 ball
 * Input is via a binary protocol where each point is a concatenation of 3
 * host-endian floats, (x,y,z). A tool point will be plotted at the
 * coordinates given.
 * At EOF, the output phase begins, writing out the depth of each pixel as a
 * float, and then exiting.
 */

#include <float.h>
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

float mmwidth, mmheight, mmdepth;
int pxwidth, pxheight;
float x_mm_px, y_mm_px;
float toolradius, toolradius_sqr;

#define FLAT 0
#define BALL 1
int toolshape;

float *map;

int seen_eof = 0;

float readfloat() {
    float f;
    if (fread(&f, sizeof(float), 1, stdin) != 1)
        seen_eof = 1;
    return f;
}

void writefloat(float f) {
    fwrite(&f, sizeof(float), 1, stdout);
}

void plot_pixel(float x, float y, float z) {
    int xpx = x / x_mm_px;
    int ypx = y / y_mm_px;

    if (xpx < 0 || ypx < 0 || xpx >= pxwidth || ypx >= pxheight)
        return;

    if (z < map[ypx*pxwidth + xpx])
        map[ypx*pxwidth + xpx] = z;
}

// plot the depth for every pixel within tool radius of (x,y)
void plot_toolpoint(float x, float y, float z) {
    for (float sy = -toolradius; sy <= toolradius; sy += y_mm_px) {
        for (float sx = -toolradius; sx <= toolradius; sx += x_mm_px) {
            float rx_sqr = sx*sx+sy*sy;
            if (rx_sqr > toolradius_sqr)
                continue;

            float zoffset = 0;
            if (toolshape == BALL)
                zoffset = toolradius - sqrtf(toolradius_sqr - rx_sqr);

            plot_pixel(x+sx, y+sy, z+zoffset);
        }
    }
}

int main(int argc, char **argv) {
    if (argc != 8) {
        fprintf(stderr, "usage: plotter mmwidth mmheight mmdepth pxwidth pxheight mmtooldiameter toolshape\n");
        return 1;
    }

    mmwidth = atof(argv[1]);
    mmheight = atof(argv[2]);
    mmdepth = atof(argv[3]);

    pxwidth = atoi(argv[4]);
    pxheight = atoi(argv[5]);

    x_mm_px = mmwidth/pxwidth;
    y_mm_px = mmheight/pxheight;

    map = malloc(sizeof(float)*pxwidth*pxheight);
    if (!map) {
        fprintf(stderr, "error: can't malloc %ld bytes\n", sizeof(float)*pxwidth*pxheight);
        return 1;
    }
    for (int y = 0; y < pxheight; y++) {
        for (int x = 0; x < pxwidth; x++) {
            map[y*pxwidth+x] = FLT_MAX;
        }
    }

    toolradius = atof(argv[6])/2;
    toolradius_sqr = toolradius*toolradius;
    if (strcmp(argv[7], "ball") == 0)
        toolshape = BALL;
    else
        toolshape = FLAT;

    while (1) {
        float x = readfloat();
        float y = readfloat();
        float z = readfloat();

        if (seen_eof)
            break;

        plot_toolpoint(x,y,z);
    }

    for (int y = 0; y < pxheight; y++) {
        for (int x = 0; x < pxwidth; x++) {
            writefloat(map[y*pxwidth+x]);
        }
    }

    return 0;
}
