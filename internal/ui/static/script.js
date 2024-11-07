document.addEventListener('DOMContentLoaded', () => {
  const fileInput = document.getElementById('fileInput');
  const convertBtn = document.getElementById('convertBtn');
  const progressContainer = document.getElementById('progress');
  const progressBar = document.getElementById('progressBar');
  const progressText = document.getElementById('progressText');
  const audioPlayer = document.createElement('audio');
  audioPlayer.controls = true;
  document.body.appendChild(audioPlayer);

  let mediaSource;
  let sourceBuffer;
  let ws = null;
  let chunks = [];
  let isPlaying = false;

  const initializeMediaSource = () => {
    mediaSource = new MediaSource();
    audioPlayer.src = URL.createObjectURL(mediaSource);

    mediaSource.addEventListener('sourceopen', () => {
      try {
        sourceBuffer = mediaSource.addSourceBuffer('audio/mp4; codecs="mp4a.40.2"');

        sourceBuffer.addEventListener('updateend', () => {
          if (chunks.length > 0 && !sourceBuffer.updating) {
            sourceBuffer.appendBuffer(chunks.shift());
          }
        });

        sourceBuffer.addEventListener('error', (e) => {
          console.error('SourceBuffer error:', e);
        });
      } catch (e) {
        console.error('Exception while adding source buffer:', e);
      }
    });

    mediaSource.addEventListener('error', (e) => {
      console.error('MediaSource error:', e);
    });
  };

  const handleStreamData = (data) => {
    try {
      const arrayBuffer = data instanceof ArrayBuffer ? data : data.buffer;
      if (sourceBuffer && !sourceBuffer.updating) {
        sourceBuffer.appendBuffer(arrayBuffer);
        isPlaying = true;
        if (audioPlayer.paused) {
          audioPlayer.play().catch(console.error);
        }
      } else {
        chunks.push(arrayBuffer);
      }
      updateProgress();
    } catch (error) {
      console.error('Error handling stream data:', error);
    }
  };

  const initializeWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;

    ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      console.log('WebSocket connection established');
    };

    ws.onmessage = (event) => {
      handleStreamData(event.data);
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('WebSocket connection closed');
      if (mediaSource.readyState === 'open') {
        mediaSource.endOfStream();
      }
    };
  };

  const sendFileInChunks = (file) => {
    const chunkSize = 64 * 1024; // 64KB chunks
    let offset = 0;

    const readAndSendChunk = () => {
      const reader = new FileReader();
      reader.onload = (e) => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(e.target.result);
          offset += e.target.result.byteLength;

          if (offset < file.size) {
            readAndSendChunk();
          } else {
            console.log('File transmission completed');
          }
        }
      };

      const slice = file.slice(offset, offset + chunkSize);
      reader.readAsArrayBuffer(slice);
    };

    readAndSendChunk();
  };

  const updateProgress = () => {
    if (audioPlayer.duration) {
      const progress = (audioPlayer.currentTime / audioPlayer.duration) * 100;
      progressBar.value = progress;
      progressText.textContent = `${Math.round(progress)}%`;
    }
  };

  convertBtn.addEventListener('click', () => {
    const file = fileInput.files[0];
    if (!file) {
      alert('Please select a WAV file');
      return;
    }

    // Reset state
    chunks = [];
    isPlaying = false;
    if (audioPlayer.src) {
      URL.revokeObjectURL(audioPlayer.src);
      audioPlayer.src = '';
    }

    // Show progress
    progressContainer.classList.remove('hidden');
    progressBar.value = 0;
    progressText.textContent = '0%';

    // Initialize streaming
    initializeMediaSource();
    initializeWebSocket();

    // Start sending file when WebSocket is ready
    ws.addEventListener('open', () => {
      sendFileInChunks(file);
    });
  });

  // Add audio player time update listener for progress
  audioPlayer.addEventListener('timeupdate', updateProgress);

  // Handle audio errors
  audioPlayer.addEventListener('error', (e) => {
    console.error('Audio playback error:', e);
  });
});