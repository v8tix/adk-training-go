/**
 * Audio Player Worklet — plays back 24kHz PCM audio chunks received from
 * /run_live.
 *
 * Ported as-is from this SDK's own reference client
 * (google.golang.org/adk/v2/examples/bidi/static/js/audio-player.js,
 * Apache 2.0) — pure Web Audio API signal processing, not this module's
 * lesson.
 */

export async function startAudioPlayerWorklet() {
    // 1. Create an AudioContext
    const audioContext = new AudioContext({
        sampleRate: 24000
    });


    // 2. Load your custom processor code
    const workletURL = new URL('./pcm-player-processor.js', import.meta.url);
    await audioContext.audioWorklet.addModule(workletURL);

    // 3. Create an AudioWorkletNode
    const audioPlayerNode = new AudioWorkletNode(audioContext, 'pcm-player-processor');

    // 4. Connect to the destination
    audioPlayerNode.connect(audioContext.destination);

    // The audioPlayerNode.port is how we send messages (audio data) to the processor
    return [audioPlayerNode, audioContext];
}
