document.addEventListener('DOMContentLoaded', () => {
  const fileInput = document.getElementById('fileInput');
  const convertBtn = document.getElementById('convertBtn');
  const progressContainer = document.getElementById('progress');
  const progressBar = document.getElementById('progressBar');
  const progressText = document.getElementById('progressText');
  const audioPlayer = document.createElement('audio');
  audioPlayer.controls = true;

  let ws = null;
  let mediaSource = null;
  let sourceBuffer = null;
  let isSourceBufferActive = false;

  // Using more compatible codec
  const mimeType = 'audio/mp4; codecs="mp4a.40.2"'; // AAC codec

  convertBtn.addEventListener('click', async () => {
    const file = fileInput.files[0];
    if (!file) {
      alert('Please select a WAV file');
      return;
    }

    // Reset previous state
    if (mediaSource) {
      if (sourceBuffer) {
        try {
          mediaSource.removeSourceBuffer(sourceBuffer);
        } catch (e) {
          console.log('Error removing source buffer:', e);
        }
      }
      mediaSource = null;
      sourceBuffer = null;
    }

    // Reset UI
    progressContainer.classList.remove('hidden');
    progressBar.value = 0;
    progressText.textContent = '0%';

    try {
      // Initialize new MediaSource
      mediaSource = new MediaSource();
      audioPlayer.src = URL.createObjectURL(mediaSource);
      document.body.appendChild(audioPlayer);

      mediaSource.addEventListener('sourceopen', setupSourceBuffer);
      mediaSource.addEventListener('sourceended', cleanupSourceBuffer);
      mediaSource.addEventListener('sourceclose', cleanupSourceBuffer);

    } catch (error) {
      console.error('Error initializing MediaSource:', error);
      alert('Error initializing audio player');
    }
  });

  function setupSourceBuffer() {
    if (!mediaSource || mediaSource.readyState !== 'open') {
      return;
    }

    try {
      sourceBuffer = mediaSource.addSourceBuffer(mimeType);
      isSourceBufferActive = true;

      sourceBuffer.addEventListener('updateend', () => {
        if (sourceBuffer && !sourceBuffer.updating) {
          updateProgress();
        }
      });

      // Initialize WebSocket after source buffer is ready
      initializeWebSocket();

    } catch (error) {
      console.error('Error setting up SourceBuffer:', error);
      cleanupSourceBuffer();
    }
  }

  function initializeWebSocket() {
    ws = new WebSocket(`ws://${window.location.host}/ws`);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      console.log('WebSocket connection opened');
      const file = fileInput.files[0];
      if (file) {
        sendFileInChunks(file);
      }
    };

    let queue = [];
    ws.onmessage = (event) => {
      if (!isSourceBufferActive || !sourceBuffer) {
        return;
      }

      try {
        if (sourceBuffer.updating) {
          queue.push(event.data);
        } else {
          appendData(event.data);
          while (queue.length > 0 && !sourceBuffer.updating) {
            appendData(queue.shift());
          }
        }
      } catch (error) {
        console.error('Error handling message:', error);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket connection closed');
      if (mediaSource && mediaSource.readyState === 'open') {
        mediaSource.endOfStream();
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      cleanupSourceBuffer();
    };
  }

  function appendData(data) {
    if (!sourceBuffer || !isSourceBufferActive) {
      return;
    }

    try {
      sourceBuffer.appendBuffer(data);
    } catch (error) {
      console.error('Error appending buffer:', error);
      if (error.name === 'InvalidStateError') {
        cleanupSourceBuffer();
      }
    }
  }

  function updateProgress() {
    if (!sourceBuffer || !isSourceBufferActive) {
      return;
    }

    try {
      if (sourceBuffer.buffered.length > 0 && audioPlayer.duration) {
        const progress = (sourceBuffer.buffered.end(0) / audioPlayer.duration) * 100;
        progressBar.value = Math.min(100, progress);
        progressText.textContent = `${Math.min(100, Math.round(progress))}%`;
      }
    } catch (error) {
      console.error('Error updating progress:', error);
    }
  }

  function cleanupSourceBuffer() {
    isSourceBufferActive = false;
    if (mediaSource && sourceBuffer) {
      try {
        mediaSource.removeSourceBuffer(sourceBuffer);
      } catch (e) {
        console.log('Error removing source buffer:', e);
      }
    }
    sourceBuffer = null;
  }

  function sendFileInChunks(file) {
    const chunkSize = 64 * 1024; // 64KB chunks
    let offset = 0;

    function readNextChunk() {
      if (!isSourceBufferActive) {
        return;
      }

      const slice = file.slice(offset, offset + chunkSize);
      const reader = new FileReader();

      reader.onload = (e) => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(e.target.result);
          offset += e.target.result.byteLength;

          if (offset < file.size) {
            readNextChunk();
          }
        }
      };

      reader.readAsArrayBuffer(slice);
    }

    readNextChunk();
  }
});