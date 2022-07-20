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

	"github.com/alexflint/go-arg"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)


var text = []string{
	"’Twas brillig",//, and the slithy toves",
	//"Did gyre and gimble in the wabe;",
	//"All mimsy were the borogoves,",
	//"And the mome raths outgrabe.",
	//"",
	//"“Beware the Jabberwock, my son!",
	//"The jaws that bite, the claws that catch!",
	//"Beware the Jubjub bird, and shun",
	//"The frumious Bandersnatch!”",
	//"",
	//"He took his vorpal sword in hand:",
	//"Long time the manxome foe he sought—",
	//"So rested he by the Tumtum tree,",
	//"And stood awhile in thought.",
	//"",
	//"And as in uffish thought he stood,",
	//"The Jabberwock, with eyes of flame,",
	//"Came whiffling through the tulgey wood,",
	//"And burbled as it came!",
	//"",
	//"One, two! One, two! and through and through",
	//"The vorpal blade went snicker-snack!",
	//"He left it dead, and with its head",
	//"He went galumphing back.",
	//"",
	//"“And hast thou slain the Jabberwock?",
	//"Come to my arms, my beamish boy!",
	//"O frabjous day! Callooh! Callay!”",
	//"He chortled in his joy.",
	//"",
	//"’Twas brillig, and the slithy toves",
	//"Did gyre and gimble in the wabe;",
	//"All mimsy were the borogoves,",
	//"And the mome raths outgrabe.",
}

var (
	dpi      = flag.Float64("dpi", 144, "screen resolution in Dots Per Inch")
	fontfile = flag.String("fontfile", "./sample.ttf", "filename of the ttf font")
	hinting  = flag.String("hinting", "none", "none | full")
	size     = flag.Float64("size", 18, "font size in points")
	spacing  = flag.Float64("spacing", 1.5, "line spacing (e.g. 2 means double spaced)")
	wonb     = flag.Bool("whiteonblack", true, "white text on a black background")
)

type Cli struct {
	Dpi float64 `arg:"--dpi" default:"72" help:"screen resolution in Dots Per Inch"`
	Pts float64 `arg:"--pts" help:"The font size in pts" default:"20"`
	Height int `arg:"--height" help:"The height of the image in pixels" default:"100"`
	Width int `arg:"--width" help:"The width of the image in pixels" default:"100"`
}

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
	ctx.SetFontSize(*size)
	ctx.SetClip(rgba.Bounds())
	ctx.SetDst(rgba)
	ctx.SetSrc(fg)
	return ctx
}

func main() {
	//flag.Parse()
	
	// Cli overrides, otherwise default value
	*size = args.Pts

	// Read the font data.
	fontBytes, err := ioutil.ReadFile(*fontfile)
	if err != nil {
		log.Println(err)
		return
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		return
	}
	*wonb = true	
	
	// create the rgba image
	rgba := image.NewRGBA(image.Rect(0, 0, 0, 0))
	
	// Initialise the ctx.
	ctx := NewCtx(f, *wonb, rgba)
	
	switch *hinting {
	default:
		ctx.SetHinting(font.HintingNone)
	case "full":
		ctx.SetHinting(font.HintingFull)
	}

	// draw the text by:
	// setting the position 
	pt := freetype.Pt(10, 10+int(ctx.PointToFixed(*size)>>6))
	
	inlineText := strings.Join(text, "")

	wordGenerator := func(out chan<- string, il string) {
		defer close(out)
		var word string
		for found := true; found; {
			found = len(il) > 0
			if ! found { break }
			word, il = string(il[:1]), string(il[1:])
			fmt.Println(word, il)

			out <- word
		}
	}

	words := make(chan string)
	go wordGenerator(words, inlineText)
	i := 0
	for word := range words {
		wordImg := image.NewRGBA(image.Rect(0, 0, args.Width, args.Height))
		wordCtx := NewCtx(f, *wonb, wordImg)
		_, err = wordCtx.DrawString(word, pt)
		
		sp2 := image.Point{rgba.Bounds().Dx(), 0}
		fmt.Println(sp2)

		r2 := image.Rectangle{sp2, sp2.Add(wordImg.Bounds().Size())}
		r := image.Rectangle{image.Point{}, r2.Max}
		
		tmp := image.NewRGBA(r)
		
		draw.Draw(tmp, rgba.Bounds(), rgba, image.Point{}, draw.Src)
		draw.Draw(tmp, r2, wordImg, sp2, draw.Src)

		*rgba = *tmp
		fmt.Println(rgba.Bounds())
		i ++
	}

	// and then using the configured context, write the string in the font.
	//_, err = ctx.DrawString(inlineText, pt)
	
	// Save that RGBA image to disk.
	outFile, err := os.Create("out.png")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = png.Encode(b, rgba)
	if err != nil {
		log.Fatal(err)
	}
	err = b.Flush()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Wrote out.png OK.")
}
