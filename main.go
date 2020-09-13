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

type ChoiceAction int

const (
	FPS          = 60
	WindowWidth  = 1024
	WindowHeight = 769
)

const (
	ActionAdd ChoiceAction = iota - 2
	ActionDelete
)

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
	Tick     Tick
	Filename int64
	Size     float64
	NumReq   int64
	DeltaT   int64
	Action   ChoiceAction
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

			// fmt.Println(record)

			tick, _ := strconv.ParseInt(record[0], 10, 64)
			filename, _ := strconv.ParseInt(record[1], 10, 64)
			size, _ := strconv.ParseFloat(record[2], 64)
			numReq, _ := strconv.ParseInt(record[3], 10, 64)
			deltaT, _ := strconv.ParseInt(record[4], 10, 64)
			action := record[5]

			var actionVal ChoiceAction

			switch {
			case action == "ADD":
				actionVal = ActionAdd
			case action == "DELETE":
				actionVal = ActionDelete
			}

			curRecord := ChoiceRecord{
				Tick:     Tick(tick),
				Filename: filename,
				Size:     size,
				NumReq:   numReq,
				DeltaT:   deltaT,
				Action:   actionVal,
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
			action := ""

			switch curRow.Action {
			case ActionAdd:
				action = "ADD"
			case ActionDelete:
				action = "DELETE"
			}

			rl.DrawText(fmt.Sprintf("%s -> %d", action, curRow.Filename), 0, 24, 24, rl.RayWhite)
		}

		// Draw
		rl.BeginDrawing()

		rl.ClearBackground(rl.Black)

		rl.DrawText(fmt.Sprintf("Tick: %d", tick), 0, 0, 24, rl.RayWhite)

		rl.EndDrawing()
	}

	rl.CloseWindow()
}
