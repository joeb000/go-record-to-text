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
var version = flag.Bool("version", false, "Print version and exits")

func addConfigPath(fileName *string) {
	s := *configDir
	s2 := *fileName
	*fileName = fmt.Sprintf("%v/%v", s, s2)
}

func main() {
	flag.Parse()

	if *version {
		astideepspeech.PrintVersions()
		return
	}

	if *configDir == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	addConfigPath(model)
	addConfigPath(alphabet)
	addConfigPath(lm)
	addConfigPath(trie)

	// Initialize DeepSpeech
	m := astideepspeech.New(*model, nCep, nContext, *alphabet, beamWidth)
	defer m.Close()
	if *lm != "" {
		m.EnableDecoderWithLM(*alphabet, *lm, *trie, lmWeight, validWordCountWeight)
	}

	inputChannels := 1
	outputChannels := 0
	sampleRate := 16000
	framesPerBuffer := make([]int16, 64)

	// init PortAudio

	portaudio.Initialize()
	defer portaudio.Terminate()

	stream, err := portaudio.OpenDefaultStream(inputChannels, outputChannels, float64(sampleRate), len(framesPerBuffer), framesPerBuffer)
	defer stream.Close()

	errCheck(err)

	//Set up stream
	streamData := astideepspeech.SetupStream(m, 0, uint(sampleRate))

	// Read
	//var d []int16

	// start stream
	errCheck(stream.Start())

	fmt.Println("RECORDING...")
	numSamples := 1100
	//Start goroutine to stream audio to deepspeech
	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan []int16, numSamples)
	go func(c chan []int16, s *astideepspeech.Stream, wg *sync.WaitGroup) {
		defer wg.Done()
		ct := 0
		for i := range c {
			ct++
			//fmt.Printf("i = %v\n", ct)
			//time.Sleep(10 * time.Millisecond)
			s.FeedAudioContent(i, uint(len(i)))
		}
		fmt.Println("\n\nDONE WITH PROCESSING NOW FINISH...")
		output := s.FinishStream()

		fmt.Printf("\n\nText: %s\n", output)

	}(c, streamData, &wg)

	for ct := 0; ct < numSamples; ct++ {
		errCheck(stream.Read())
		//send copy to chan
		copyFPB := make([]int16, len(framesPerBuffer))
		copy(copyFPB, framesPerBuffer)
		c <- copyFPB

		fmt.Printf("\n Recording... Say something to your microphone! [%v]  |  len= %v | cap= %v", ct, len(c), cap(c))

	}

	close(c)

	wg.Wait()

}
