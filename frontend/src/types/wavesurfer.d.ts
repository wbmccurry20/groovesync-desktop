declare module 'wavesurfer.js' {
    interface WaveSurferOptions {
      container: string | HTMLElement;
      waveColor?: string;
      progressColor?: string;
      height?: number;
      barWidth?: number;
      barGap?: number;
    }
  
    class WaveSurfer {
      static create(options: WaveSurferOptions): WaveSurfer;
      load(url: string): void;
      destroy(): void;
    }
  
    export default WaveSurfer;
  }