declare module 'qrcode.react';
declare module 'react-modal-image';
declare module 'react-dom/client';
declare module 'react-signature-canvas' {
  import React from 'react';
  interface SignatureCanvasProps {
    penColor?: string;
    canvasProps?: React.CanvasHTMLAttributes<HTMLCanvasElement>;
    onBegin?: () => void;
    onEnd?: () => void;
    [key: string]: any;
  }
  class SignatureCanvas extends React.Component<SignatureCanvasProps> {
    clear(): void;
    isEmpty(): boolean;
    toDataURL(type?: string): string;
    fromDataURL(dataURL: string): void;
    getCanvas(): HTMLCanvasElement;
    getTrimmedCanvas(): HTMLCanvasElement;
    off(): void;
    on(): void;
  }
  export default SignatureCanvas;
}
declare module 'mic-recorder-to-mp3' {
  const MicRecorder: any;
  export default MicRecorder;
}
declare module '*.png';
declare module '*.mp3';
