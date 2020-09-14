package main

import (
	"compress/gzip"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	FPS          = 60
	WindowWidth  = 1024
	WindowHeight = 769
)

type ActionOrEventType string

type Tick int

func (tick *Tick) Update(diff int) {
	*tick += Tick(diff)
	if *tick < 0 {
		*tick = Tick(0)
	}
}

var (
	filename  string
	csvHeader = "tick,filename,size,num req,delta t,action"
)

func init() {
	// Tie the command-line flag to the intervalFlag variable and
	// set a usage message.
	flag.StringVar(&filename, "filename", "", "comma-separated list of files to load")
}

type ChoiceRecord struct {
	Tick          Tick
	ActionOrEvent ActionOrEventType
	CacheSize     float64
	CacheCapacity float64
	Filename      int64
	Size          float64
	NumReq        int64
	DeltaT        int64
}

func recordGenerator(csvReader *csv.Reader, curFile *os.File) chan ChoiceRecord { //nolint:ignore,funlen
	channel := make(chan ChoiceRecord, FPS)

	go func() {
		defer close(channel)
		defer curFile.Close()

		for {
			record, err := csvReader.Read()

			if err == io.EOF {
				break
			}

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(record)

			tick, _ := strconv.ParseInt(record[0], 10, 64)
			action := ActionOrEventType(record[1])
			cacheSize, _ := strconv.ParseFloat(record[2], 64)
			cacheCapacity, _ := strconv.ParseFloat(record[3], 64)
			filename, _ := strconv.ParseInt(record[4], 10, 64)
			size, _ := strconv.ParseFloat(record[5], 64)
			numReq, _ := strconv.ParseInt(record[6], 10, 64)
			deltaT, _ := strconv.ParseInt(record[7], 10, 64)

			curRecord := ChoiceRecord{
				Tick:          Tick(tick),
				ActionOrEvent: action,
				CacheSize:     cacheSize,
				CacheCapacity: cacheCapacity,
				Filename:      filename,
				Size:          size,
				NumReq:        numReq,
				DeltaT:        deltaT,
			}

			// fmt.Println(curRecord)

			channel <- curRecord
		}
	}()

	return channel
}

func OpenSimFile(filePath string) chan ChoiceRecord {
	fileExt := path.Ext(filePath)

	var iterator chan ChoiceRecord

	curFile, errOpenFile := os.Open(filePath)

	if errOpenFile != nil {
		panic(errOpenFile)
	}

	switch fileExt {
	case ".gz", ".gzip":
		// Create new reader to decompress gzip.
		curCsv, errReadGz := gzip.NewReader(curFile)

		if errReadGz != nil {
			panic(errReadGz)
		}

		csvReader := csv.NewReader(curCsv)
		// Discar header
		_, errCSVRead := csvReader.Read()

		if errCSVRead != nil {
			panic(errCSVRead)
		}

		iterator = recordGenerator(csvReader, curFile)
	default:
		csvReader := csv.NewReader(curFile)
		// Discar header
		_, errCSVRead := csvReader.Read()

		if errCSVRead != nil {
			panic(errCSVRead)
		}

		iterator = recordGenerator(csvReader, curFile)
	}

	return iterator
}

func main() {
	flag.Parse()

	curFile := OpenSimFile(filename)

	rl.InitWindow(WindowWidth, WindowHeight, "SmartCache - Simulation viewer")

	rl.SetTargetFPS(FPS)

	var (
		tick   Tick
		curRow ChoiceRecord
		buffer = make(map[Tick]ChoiceRecord)
	)

	curRow = <-curFile
	buffer[curRow.Tick] = curRow

	for !rl.WindowShouldClose() {
		// Inputs
		if rl.IsKeyDown(rl.KeyRight) {
			tick.Update(1)
		}

		if rl.IsKeyDown(rl.KeyLeft) {
			tick.Update(-1)
		}

		if curRow.Tick < tick {
			for curRow.Tick < tick {
				curRow = <-curFile
				buffer[curRow.Tick] = curRow
			}
		}

		curRow, inBuffer := buffer[tick]
		if inBuffer {
			rl.DrawText(fmt.Sprintf("%s -> %d", curRow.ActionOrEvent, curRow.Filename), 0, 24, 24, rl.RayWhite)
		}

		// Draw
		rl.BeginDrawing()

		rl.ClearBackground(rl.Black)

		rl.DrawRectangle(142, 142, 1, 1, rl.Red)
		rl.DrawRectangleLines(160, 320, 80, 60, rl.Orange)

		rl.DrawText(fmt.Sprintf("Tick: %d", tick), 0, 0, 24, rl.RayWhite)

		rl.EndDrawing()
	}

	rl.CloseWindow()
}
