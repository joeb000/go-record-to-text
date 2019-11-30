# go-record-to-text

The go-record-to-text is a command line client which records audio from the default input audio device and streams the raw audio data into a speech to text engine.

For audio input, this project uses [gordonklaus's go wrapper](https://github.com/gordonklaus/portaudio) around the C++ [portaudio library](http://www.portaudio.com/)

For speech to text, this project uses [Mozilla's Deepspeech](https://github.com/mozilla/DeepSpeech) engine with a pre-trained model, more specifically via asicode's [golang deepspeech wrapper](https://github.com/asticode/go-astideepspeech)

To get this working locally, you must follow the instructions for installing Deepspeech [here](https://github.com/asticode/go-astideepspeech#install-deepspeech).