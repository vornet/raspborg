package main

import (
	"github.com/disintegration/gift"
	"github.com/blackjack/webcam"
	"os"
	"fmt"
	"bytes"
	"image/jpeg"
	"net/http"
	"image"
	"image/color"
	"image/draw"
	"log"
	"strconv"
	"sync"
)

var (
	frameImage image.Image
	frameImageMutex = &sync.Mutex{}
)


func webcamFrameGrabber() {
	// ...
	cam, err := webcam.Open("/dev/video0") // Open webcam
	if err != nil { panic(err.Error()) }
	defer cam.Close()
	// ...
	// Setup webcam image format and frame size here (see examples or documentation)
	// ...

	format_desc := cam.GetSupportedFormats()
	var formats []webcam.PixelFormat
	for f := range format_desc {
		formats = append(formats, f)
		fmt.Printf("%v\n", format_desc[f])
	}

	format := formats[0]
	// size := cam.GetSupportedFrameSizes(format)[0]

	_, _, _, err = cam.SetImageFormat(format, uint32(1920), uint32(1080))
	if err != nil { panic(err.Error()) }

	err = cam.StartStreaming()
	if err != nil { panic(err.Error()) }

	timeout := uint32(5) // 5 seconds
	for {
		err = cam.WaitForFrame(timeout)

		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			fmt.Fprint(os.Stderr, err.Error())
			continue
		default:
			panic(err.Error())
		}

		frame, err := cam.ReadFrame()
		if len(frame) != 0 {
			// Process frame
			reader := bytes.NewReader(frame)
			frameImageMutex.Lock()
			frameImage, err = jpeg.Decode(reader)
			frameImageMutex.Unlock()
			if err == nil {
				fmt.Printf("Got frame! %v, %v\n", frameImage.Bounds().Size().X, frameImage.Bounds().Size().Y)
				// toimg, _ := os.Create("testing.jpg")
				// jpeg.Encode(toimg, image, &jpeg.Options{jpeg.DefaultQuality})
				// toimg.Close()
			}

		} else if err != nil {
			panic(err.Error())
		}
	}
}

func main() {
	// Start webcam frame grabbing
	go webcamFrameGrabber()

	http.HandleFunc("/cam/", camHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// writeImage encodes an image 'img' in jpeg format and writes it into ResponseWriter.
func writeImage(w http.ResponseWriter, img *image.Image) {

	buffer := new(bytes.Buffer)
	if err := jpeg.Encode(buffer, *img, nil); err != nil {
		log.Println("unable to encode image.")
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println("unable to write image.")
	}
}

func camHandler(w http.ResponseWriter, r *http.Request) {
	frameImageMutex.Lock()

	g := gift.New(
		gift.Sobel(),
	)


	m := image.NewRGBA(g.Bounds(frameImage.Bounds()))

	g.Draw(m, frameImage)

	blue := color.RGBA{0, 0, 255, 255}
	draw.Draw(m, image.Rect(0,0,100,100), &image.Uniform{blue}, image.ZP, draw.Src)

	var img image.Image = m
	writeImage(w, &img)

	frameImageMutex.Unlock()
}