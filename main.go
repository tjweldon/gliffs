// Copyright 2010 The Freetype-Go Authors. All rights reserved.
// Use of this source code is governed by your choice of either the
// FreeType License or the GNU General Public License version 2 (or
// any later version), both of which can be found in the LICENSE file.

//go:build example
// +build example

//
// This build tag means that "go install github.com/golang/freetype/..."
// doesn't install this example program. Use "go run main.go" to run it or "go
// install -tags=example" to install it.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

var text = strings.Join([]string{
	"’Twas brillig, and the slithy toves",
	"Did gyre and gimble in the wabe;",
	"All mimsy were the borogoves,",
	"And the mome raths outgrabe.",
	"“Beware the Jabberwock, my son!",
	"The jaws that bite, the claws that catch!",
	"Beware the Jubjub bird, and shun",
	"The frumious Bandersnatch!”",
	"He took his vorpal sword in hand:",
	"Long time the manxome foe he sought—",
	"So rested he by the Tumtum tree,",
	"And stood awhile in thought.",
	"And as in uffish thought he stood,",
	"The Jabberwock, with eyes of flame,",
	"Came whiffling through the tulgey wood,",
	"And burbled as it came!",
	"One, two! One, two! and through and through",
	"The vorpal blade went snicker-snack!",
	"He left it dead, and with its head",
	"He went galumphing back.",
	"“And hast thou slain the Jabberwock?",
	"Come to my arms, my beamish boy!",
	"O frabjous day! Callooh! Callay!”",
	"He chortled in his joy.",
	"’Twas brillig, and the slithy toves",
	"Did gyre and gimble in the wabe;",
	"All mimsy were the borogoves,",
	"And the mome raths outgrabe.",
}, " ")

var (
	dpi      = flag.Float64("dpi", 72, "screen resolution in Dots Per Inch")
	fontfile = flag.String("fontfile", "./sample.ttf", "filename of the ttf font")
	hinting  = flag.String("hinting", "full", "none | full")
	spacing  = flag.Float64("spacing", 1.5, "line spacing (e.g. 2 means double spaced)")
)

type Cli struct {
	Dpi    float64 `arg:"--dpi" default:"72" help:"screen resolution in Dots Per Inch"`
	Pts    float64 `arg:"--pts" help:"The font size in pts" default:"20"`
	Height int     `arg:"--height" help:"The height of the image in pixels" default:"0"`
	Width  int     `arg:"--width" help:"The width of the image in pixels" default:"0"`
	Light  bool    `arg:"--light" help:"Run in light mode" default:"false"`
}

func (c Cli) GetDims() (w, h int) { return c.Width, c.Height }
func (c Cli) GetLightMode() bool  { return c.Light }

var args = func() Cli {
	a := &Cli{}
	arg.MustParse(a)
	return *a
}()

func NewCtx(f *truetype.Font, whiteOnBlack bool, rgba draw.Image) (ctx *freetype.Context) {
	// Initialize the ctx.
	fg, bg := image.Black, image.White
	if whiteOnBlack {
		fg, bg = image.White, image.Black
	}
	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
	ctx = freetype.NewContext()
	ctx.SetDPI(*dpi)
	ctx.SetFont(f)
	ctx.SetFontSize(args.Pts)
	ctx.SetClip(rgba.Bounds())
	ctx.SetDst(rgba)
	ctx.SetSrc(fg)
	ctx.SetHinting(font.HintingFull)
	return ctx
}

type TypeFace struct {
	Font *truetype.Font
	Face font.Face
}

func GenWords(text string) <-chan string {
	work := func(out chan<- string, txt string) {
		defer close(out)
		for found := true; found; {
			var word, rest string
			word, rest, found = strings.Cut(txt, " ")
			txt = string(rest)
			out <- string(word)
		}
	}

	res := make(chan string)
	go work(res, text)
	return res
}

func main() {
	typeface, err := LoadFont(*fontfile, args.Pts)
	if err != nil {
		log.Fatal(err)
	}
	if args.Height == 0 {
		args.Height = int(typeface.Face.Metrics().Height)>>6 + int(typeface.Face.Metrics().Descent)>>6
	}
	if args.Width == 0 {
		w, ok := typeface.Face.GlyphAdvance('$')
		if !ok {
			log.Fatal("Could not get glyph width")
		}
		args.Width = int(w) >> 6
	}
	words := GenWords(text)

	results := GenImages(words, args, args, typeface, args.Pts)
	for img := range results {
		if err := SaveImg(img, "out.png"); err != nil {
			log.Fatal(err)
		}
	}
}

type HasDims interface {
	GetDims() (w, h int)
}

type HasLightMode interface {
	GetLightMode() bool
}

func LoadFont(path string, size float64) (typeface *TypeFace, err error) {
	// everything is declared at this point, so returning if
	// err != nil will return nil, err
	var fontBytes []byte

	// Read the font data.
	fontBytes, err = ioutil.ReadFile(*fontfile)
	if err != nil {
		return
	}

	// then parse it
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return
	}

	face := truetype.NewFace(f, &truetype.Options{Size: size})

	return &TypeFace{Font: f, Face: face}, nil
}

func LefToRightConcat(imgs ...draw.Image) (concat draw.Image, err error) {
	if len(imgs) == 0 {
		return nil, fmt.Errorf("LefToRightConcat: Must supply at least one image")
	}
	cellWidth := imgs[0].Bounds().Dx()

	width := 0
	for _, nxt := range imgs {
		width += nxt.Bounds().Dx()
	}
	concat = image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{
				width,
				imgs[0].Bounds().Dy(),
			},
		},
	)

	for i, nxt := range imgs {
		draw.Draw(
			concat,
			nxt.Bounds().Bounds().Add(image.Point{i * cellWidth, 0}),
			nxt,
			image.Point{0, 0},
			draw.Src,
		)
	}

	return
}

func GenImages(text <-chan string, dims HasDims, light HasLightMode, typeface *TypeFace, pts float64) <-chan image.Image {
	wonb := !light.GetLightMode()
	work := func(res chan<- image.Image, txt <-chan string, dark bool) {
		defer close(res)
		imgs := []draw.Image{}
		for x := range text {
			for _, glyph := range x {
				// create the rgba image
				glyphImg := image.NewRGBA(image.Rect(0, 0, args.Width, args.Height))

				// Initialise the ctx.
				ctx := NewCtx(typeface.Font, dark, glyphImg)
				m := typeface.Face.Metrics()
				pixelHeight := int(m.Height) >> 6
				
				// set the positioning
				pt := freetype.Pt(0, pixelHeight)
				
				// draw
				ctx.DrawString(string(glyph), pt)
				imgs = append(imgs, glyphImg)
			}
			// concatenate the images and...
			concat, err := LefToRightConcat(imgs...)
			if err != nil {
				log.Fatal(err)
			}
			// send concat!
			res <- concat
			imgs = []draw.Image{}
			time.Sleep(200 * time.Millisecond)
		}
		

	}

	out := make(chan image.Image)
	go work(out, text, wonb)

	return out
}

func SaveImg(rgba image.Image, path string) error {
	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = png.Encode(b, rgba)
	if err != nil {
		return err
	}
	err = b.Flush()
	if err != nil {
		return err
	}
	log.Println("Wrote out.png OK.")

	return nil
}
