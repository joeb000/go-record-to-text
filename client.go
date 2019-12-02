package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/gordonklaus/portaudio"

	"github.com/asticode/go-astideepspeech"
)

func errCheck(err error) {

	if err != nil {
		panic(err)
	}
}

// Constants
const (
	beamWidth            = 500
	nCep                 = 26
	nContext             = 9
	lmWeight             = 0.75
	validWordCountWeight = 1.85
)

var configDir = flag.String("configDir", "", "Path to config directory for the DeepSpeech Model files")
var model = flag.String("model", "output_graph.pbmm", "File name of the model (protocol buffer binary file)")
var alphabet = flag.String("alphabet", "alphabet.txt", "File name of the configuration file specifying the alphabet used by the network")
var lm = flag.String("lm", "lm.binary", "File name of the language model binary file")
var trie = flag.String("trie", "trie", "File name of the language model trie file created with native_client/generate_trie")

func addConfigPath(fileName *string) {
	*fileName = fmt.Sprintf("%v/%v", *configDir, *fileName)
}

func configureFlags() {
	flag.Parse()

	if *configDir == "" {
		fmt.Println("No config Directory Specified")
		flag.PrintDefaults()
		os.Exit(0)
	}

	addConfigPath(model)
	addConfigPath(alphabet)
	addConfigPath(lm)
	addConfigPath(trie)
}

func main() {

	configureFlags()

	// Initialize DeepSpeech
	m := astideepspeech.New(*model, nCep, nContext, *alphabet, beamWidth)
	defer m.Close()
	if *lm != "" {
		m.EnableDecoderWithLM(*alphabet, *lm, *trie, lmWeight, validWordCountWeight)
	}

	// init PortAudio
	err := portaudio.Initialize()
	errCheck(err)
	defer portaudio.Terminate()

	inputChannels := 1
	outputChannels := 0
	sampleRate := 16000
	framesPerBuffer := make([]int16, 64)

	stream, err := portaudio.OpenDefaultStream(inputChannels, outputChannels, float64(sampleRate), len(framesPerBuffer), framesPerBuffer)
	errCheck(err)
	defer stream.Close()

	//Set up stream
	dsStream := astideepspeech.SetupStream(m, 0, uint(sampleRate))

	// start portaudio input stream
	errCheck(stream.Start())

	fmt.Println("RECORDING...")

	numSamples := 2000

	// Make a channel to send samples through
	c := make(chan []int16, 100)

	// create wait group
	var wg sync.WaitGroup
	wg.Add(1)

	//Start goroutine to stream audio to deepspeech
	go func() {
		defer wg.Done()

		for i := range c {
			dsStream.FeedAudioContent(i, uint(len(i)))
		}

		fmt.Println("\n\nDONE WITH PROCESSING NOW FINISH...")

		// Call finish stream to get the text output from deepspeech
		output := dsStream.FinishStream()

		fmt.Printf("\n\nText: %s\n", output)

	}()

	// important that we minimize operations inside the loop so we don't overflow audio input buffer
	for ct := 0; ct < numSamples; ct++ {
		errCheck(stream.Read())
		//send copy to chan
		copyFPB := make([]int16, len(framesPerBuffer))
		copy(copyFPB, framesPerBuffer)
		c <- copyFPB

		fmt.Printf("\r Recording... Say something to your microphone! [%v]  |  len= %v | cap= %v", ct, len(c), cap(c))

	}

	close(c)

	wg.Wait()

}
