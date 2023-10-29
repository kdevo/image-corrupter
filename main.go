package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"math/rand"
	"os"
	"time"
)

var seededRand = rand.New(rand.NewSource(1))

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

// wrap forces x to stay in [0, b) range. x is assumed to be in [-b,2*b) range
func wrap(x, b int) int {
	if x < 0 {
		return x + b
	}
	if x >= b {
		return x - b
	}
	return x
}

// offset gets normally distributed (rounded to int) value with the specified std. dev.
func offset(stddev float64) int {
	sample := seededRand.NormFloat64() * stddev
	return int(sample)
}

// brighten the color safely, i.e., by simultaneously reducing contrast
func brighten(r uint8, add uint8) uint8 {
	r32, add32 := uint32(r), uint32(add)
	return uint8(r32 - r32*add32/255 + add32)
}

func main() {
	// command line parsing
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [input] [output]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   or: %s [options] - (for stdin+stdout processing)\n", os.Args[0])
		flag.PrintDefaults()
	}
	magPtr := flag.Float64("mag", 7.0, "dissolve blur strength")
	blockHeightPtr := flag.Int("bheight", 10, "average distorted block height")
	blockOffsetPtr := flag.Float64("boffset", 30., "distorted block offset strength")
	strideMagPtr := flag.Float64("stride", 0.1, "distorted block stride strength")

	lagPtr := flag.Float64("lag", 0.005, "per-channel scanline lag strength")
	lrPtr := flag.Float64("lr", -7, "initial red scanline lag")
	lgPtr := flag.Float64("lg", 0, "initial green scanline lag")
	lbPtr := flag.Float64("lb", 3, "initial blue scanline lag")
	stdOffsetPtr := flag.Float64("stdoffset", 10, "std. dev. of red-blue channel offset (non-destructive)")
	addPtr := flag.Int("add", 37, "additional brightness control (0-255)")

	meanAbberPtr := flag.Int("meanabber", 10, "mean chromatic abberation offset")
	stdAbberPtr := flag.Float64("stdabber", 10, "std. dev. of chromatic abberation offset (lower values induce longer trails)")

	seedPtr := flag.Int64("seed", -1, "random seed. set to -1 if you want to generate it from time. the old version has used seed=1")

	flag.Parse()

	if *seedPtr == -1 {
		seededRand = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	} else if *seedPtr != 1 {
		seededRand = rand.New(rand.NewSource(*seedPtr))
	}

	// flag.Args() contain all non-option arguments, i.e., our input and output files
	var reader *os.File
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(2)
	} else if flag.Args()[0] == "-" {
		// stdin/stdout processing
		reader = os.Stdin
	} else if len(flag.Args()) == 2 {
		var err error
		reader, err = os.Open(flag.Args()[0])
		check(err)
	} else {
		flag.Usage()
		os.Exit(2)
	}
	m, err := png.Decode(reader)
	check(err)
	reader.Close()

	// trying to obtain raw pointers to color data, since .At(), .Set() are very slow
	mRawStride, mRawPix := 0, []uint8(nil)

	switch m := m.(type) {
	case *image.NRGBA:
		mRawStride = m.Stride
		mRawPix = m.Pix
	case *image.RGBA:
		mRawStride = m.Stride
		mRawPix = m.Pix
	default:
		log.Fatal("unknown image type")
	}

	b := m.Bounds()

	// first stage is dissolve+block corruption
	newImg := image.NewNRGBA(b)
	lineOff := 0
	stride := 0.
	yset := 0
	mag := *magPtr
	bHeight := *blockHeightPtr
	bOffset := *blockOffsetPtr
	strideMag := *strideMagPtr
	for y := 0; y < b.Max.Y; y++ {
		for x := 0; x < b.Max.X; x++ {
			// Every bHeight lines in average a new distorted block begins
			if seededRand.Intn(bHeight*b.Max.X) == 0 {
				lineOff = offset(bOffset)
				stride = seededRand.NormFloat64() * strideMag
				yset = y
			}
			// at the line where the block has begun, we don't want to offset the image
			// so strideOff is 0 on the block's line
			strideOff := int(stride * float64(y-yset))

			// offset is composed of the blur, block offset, and skew offset (stride)
			offx := offset(mag) + lineOff + strideOff
			offy := offset(mag)

			// copy the corresponding pixel (4 bytes) to the new image
			srcIdx := mRawStride*wrap(y+offy, b.Max.Y) + 4*wrap(x+offx, b.Max.X)
			dstIdx := newImg.Stride*y + 4*x

			copy(newImg.Pix[dstIdx:dstIdx+4], mRawPix[srcIdx:srcIdx+4])
		}
	}

	// second stage is adding per-channel scan inconsistency and brightening
	newImg1 := image.NewNRGBA(b)

	lr, lg, lb := *lrPtr, *lgPtr, *lbPtr
	lag := *lagPtr
	add := uint8(*addPtr)
	stdOffset := *stdOffsetPtr
	for y := 0; y < b.Max.Y; y++ {
		for x := 0; x < b.Max.X; x++ {
			lr += seededRand.NormFloat64() * lag
			lg += seededRand.NormFloat64() * lag
			lb += seededRand.NormFloat64() * lag
			offx := offset(stdOffset)

			// obtain source pixel base offsets. red/blue border is also smoothed by offx
			raIdx := newImg.Stride*y + 4*wrap(x+int(lr)-offx, b.Max.X)
			gIdx := newImg.Stride*y + 4*wrap(x+int(lg), b.Max.X)
			bIdx := newImg.Stride*y + 4*wrap(x+int(lb)+offx, b.Max.X)

			// pixels are stored in (r, g, b, a) order in memory
			r := newImg.Pix[raIdx]
			a := newImg.Pix[raIdx+3]
			g := newImg.Pix[gIdx+1]
			b := newImg.Pix[bIdx+2]

			r, g, b = brighten(r, add), brighten(g, add), brighten(b, add)

			// copy the corresponding pixel (4 bytes) to the new image
			dstIdx := newImg1.Stride*y + 4*x
			copy(newImg1.Pix[dstIdx:dstIdx+4], []uint8{r, g, b, a})
		}
	}

	// third stage is to add chromatic abberation+chromatic trails
	// (trails happen because we're changing the same image we process)
	meanAbber := *meanAbberPtr
	stdAbber := *stdAbberPtr
	for y := 0; y < b.Max.Y; y++ {
		for x := 0; x < b.Max.X; x++ {
			offx := meanAbber + offset(stdAbber) // lower offset arg = longer trails

			// obtain source pixel base offsets. only red and blue are distorted
			raIdx := newImg1.Stride*y + 4*wrap(x+offx, b.Max.X)
			gIdx := newImg1.Stride*y + 4*x
			bIdx := newImg1.Stride*y + 4*wrap(x-offx, b.Max.X)

			// pixels are stored in (r, g, b, a) order in memory
			r := newImg1.Pix[raIdx]
			a := newImg1.Pix[raIdx+3]
			g := newImg1.Pix[gIdx+1]
			b := newImg1.Pix[bIdx+2]

			// copy the corresponding pixel (4 bytes) to the SAME image. this gets us nice colorful trails
			dstIdx := newImg1.Stride*y + 4*x
			copy(newImg1.Pix[dstIdx:dstIdx+4], []uint8{r, g, b, a})
		}
	}

	// write the image
	var writer *os.File
	if flag.Args()[0] == "-" {
		// stdin/stdout processing
		writer = os.Stdout
	} else {
		writer, err = os.Create(flag.Args()[1])
		check(err)
	}
	e := png.Encoder{CompressionLevel: png.NoCompression}
	e.Encode(writer, newImg1)
	writer.Close()
}
