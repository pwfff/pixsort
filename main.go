package pixsort

import (
	"bufio"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"
)

type sortRow struct {
	start int
	end   int
}

type YSorter []color.Color

func (r YSorter) Len() int      { return len(r) }
func (r YSorter) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r YSorter) Less(i, j int) bool {
	r1, g1, b1, _ := r[i].RGBA()
	r2, g2, b2, _ := r[j].RGBA()
	return (r1 + g1 + b1) < (r2 + g2 + b2)
}

func getDrawableImage(src image.Image) *image.RGBA {
	b := src.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)
	return rgba
}

func getImage(path string) (*image.RGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)

	if err != nil {
		return nil, err
	}

	return getDrawableImage(img), nil
}

func fitImage(src, dest *image.RGBA) *image.RGBA {
	bounds := src.Bounds()
	srcWidth, srcHeight := uint(bounds.Max.X), uint(bounds.Max.Y)
	srcRatio := float64(srcWidth) / float64(srcHeight)

	bounds = dest.Bounds()
	destWidth, destHeight := uint(bounds.Max.X), uint(bounds.Max.Y)

	var fit image.Image
	if srcRatio > 1.0 {
		fit = resize.Resize(destWidth, 0, src, resize.Lanczos3)
	} else {
		fit = resize.Resize(0, destHeight, src, resize.Lanczos3)
	}

	return getDrawableImage(fit)
}

func getPixels(img *image.RGBA) [][]color.Color {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]color.Color
	for y := 0; y < height; y++ {
		var row []color.Color
		for x := 0; x < width*4; x += 4 {
			index := y * width * 4
			row = append(row, color.RGBA{img.Pix[index+x], img.Pix[index+x+1], img.Pix[index+x+2], img.Pix[index+x+3]})
		}
		pixels = append(pixels, row)
	}

	return pixels
}

func main() {
	//rand.Seed(time.Now().UTC().UnixNano())
	baseImage, err := getImage("/home/pwf/go/city.jpg")
	if err != nil {
		log.Fatal(err)
	}
	basePixels := getPixels(baseImage)

	maskImage, err := getImage("/home/pwf/go/mask.png")
	if err != nil {
		log.Fatal(err)
	}
	maskImage = fitImage(maskImage, baseImage)

	maskRows := GetMaskRows(maskImage)

	DoSort(baseImage, basePixels, maskRows)

	f, err := os.Create("/home/pwf/go/out.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	jpeg.Encode(w, baseImage, nil)

	//log.Print(basePixels[1][1].RGBA())
	//log.Print(maskRows[0])
}

func GetMaskRows(maskImage *image.RGBA) [][]sortRow {
	maskBounds := maskImage.Bounds()
	width, height := maskBounds.Max.X, maskBounds.Max.Y

	var maskRows [][]sortRow
	for y := 0; y < height; y++ {
		var rows []sortRow
		var start int
		var inRow bool

		for x := 0; x < width; x++ {
			index := y*width*4 + 3
			a := maskImage.Pix[index]
			if a == 0 {
				if inRow {
					rows = append(rows, sortRow{start, x})
					inRow = false
				}
			} else {
				if !inRow {
					start = x
					inRow = true
				} else {
					if rand.Float64() > .95 {
						rows = append(rows, sortRow{start, x})
						inRow = false
					}
				}
			}
		}
		maskRows = append(maskRows, rows)
	}

	return maskRows
}

func DoSort(baseImage *image.RGBA, basePixels [][]color.Color, maskRows [][]sortRow) {
	var wg sync.WaitGroup
	for y := range maskRows {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			for i := range maskRows[y] {
				row := maskRows[y][i]

				sort.Sort(YSorter(basePixels[y][row.start:row.end]))

				for x := row.start; x < row.end; x++ {
					baseImage.Set(x, y, basePixels[y][x])
				}
			}
			//log.Print(y)
		}(y)
	}
	wg.Wait()
}
