// Ported as-is from this SDK's own reference client
// (google.golang.org/adk/v2/examples/bidi/static/js/pcm-recorder-processor.js,
// Apache 2.0) — pure Web Audio API signal processing, not this module's lesson.
class PCMProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
  }

  process(inputs, outputs, parameters) {
    if (inputs.length > 0 && inputs[0].length > 0) {
      // Use the first channel
      const inputChannel = inputs[0][0];
      // Copy the buffer to avoid issues with recycled memory
      const inputCopy = new Float32Array(inputChannel);
      this.port.postMessage(inputCopy);
    }
    return true;
  }
}

registerProcessor("pcm-recorder-processor", PCMProcessor);
