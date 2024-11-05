document.addEventListener('DOMContentLoaded', () => {
  const fileInput = document.getElementById('fileInput');
  const convertBtn = document.getElementById('convertBtn');
  const progressContainer = document.getElementById('progress');
  const progressBar = document.getElementById('progressBar');
  const progressText = document.getElementById('progressText');
  const resultContainer = document.getElementById('result');
  const downloadBtn = document.getElementById('downloadBtn');

  // Use relative WebSocket URL
  const WS_URL = `ws://${window.location.host}/ws`;

  let ws = null;
  let flacChunks = [];

  convertBtn.addEventListener('click', () => {
    const file = fileInput.files[0];
    if (!file) {
      alert('Please select a WAV file');
      return;
    }

    // Reset UI
    progressContainer.classList.remove('hidden');
    resultContainer.classList.add('hidden');
    flacChunks = [];

    // Establish WebSocket connection
    ws = new WebSocket(WS_URL);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      console.log('WebSocket connection opened');
      sendFileInChunks(file);
    };

    ws.onmessage = (event) => {
      console.log('Received FLAC chunk:', event.data.byteLength);
      flacChunks.push(event.data);
      updateProgress(flacChunks.length);
    };

    ws.onclose = () => {
      console.log('WebSocket connection closed');
      if (flacChunks.length > 0) {
        progressContainer.classList.add('hidden');
        resultContainer.classList.remove('hidden');
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      alert('WebSocket connection failed');
    };
  });

  downloadBtn.addEventListener('click', () => {
    const blob = new Blob(flacChunks, { type: 'audio/flac' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'converted.flac';
    a.click();
    URL.revokeObjectURL(url);
  });

  function sendFileInChunks(file) {
    const chunkSize = 4096;
    let offset = 0;

    function readSlice() {
      const slice = file.slice(offset, offset + chunkSize);
      const reader = new FileReader();

      reader.onload = (event) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(event.target.result);
          offset += event.target.result.byteLength;

          if (offset < file.size) {
            readSlice();
          } else {
            ws.close();
          }
        }
      };

      reader.readAsArrayBuffer(slice);
    }

    readSlice();
  }

  function updateProgress(chunkCount) {
    const totalChunks = Math.ceil(fileInput.files[0].size / 4096);
    const progress = Math.min((chunkCount / totalChunks) * 100, 100);

    progressBar.value = progress;
    progressText.textContent = `${progress.toFixed(2)}%`;
  }
});