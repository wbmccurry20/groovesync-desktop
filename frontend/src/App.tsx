import { useState, useEffect } from 'react';
import { StartDownload, ExportToRekordbox } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './App.css';

interface Progress {
    current: number;
    total: number;
}

interface Track {
    url: string;
    title: string;
}

function App() {
    const [url, setUrl] = useState('');
    const [name, setName] = useState('');
    const [format, setFormat] = useState('wav');
    const [dir, setDir] = useState('');
    const [status, setStatus] = useState('Ready to start downloading...');
    const [progress, setProgress] = useState<Progress>({ current: 0, total: 0 });
    const [tracks, setTracks] = useState<Track[]>([]);
    const [isDownloading, setIsDownloading] = useState(false);
    const [downloadHistory, setDownloadHistory] = useState<string[]>([]);
    const [exportPath, setExportPath] = useState<string>(''); // For Rekordbox export path

    useEffect(() => {
        EventsOn('status', (status: string) => {
            setStatus(status);
            if (status.includes("downloaded successfully")) {
                setIsDownloading(false);
                setDownloadHistory(prev => [`${name} - ${new Date().toLocaleTimeString()}`, ...prev.slice(0, 4)]);
            } else if (status.includes("failed")) {
                setIsDownloading(false);
            }
        });
        EventsOn('progress', (progress: Progress) => setProgress(progress));
        EventsOn('tracks', (tracks: Track[]) => setTracks(tracks));
        EventsOn('exportCompleted', (data: { path: string }) => {
            setExportPath(data.path);
            setStatus(`Exported to Rekordbox successfully! File: ${data.path}`);
        });
    }, [name]);

    const handleStartDownload = async () => {
        if (!url || !name) {
            setStatus('Error: Playlist URL and Name are required');
            return;
        }
        setIsDownloading(true);
        setProgress({ current: 0, total: 0 });
        setExportPath('');
        try {
            await StartDownload(url, name, format, dir);
        } catch (error) {
            setStatus('Download failed. Check logs.');
            setIsDownloading(false);
            console.error('Download error:', error);
        }
    };

    const handleExportToRekordbox = async () => {
        if (!name) {
            setStatus('Error: Playlist Name is required for export');
            return;
        }
        try {
            await ExportToRekordbox(name);
        } catch (error) {
            setStatus('Export to Rekordbox failed. Check logs.');
            console.error('Export error:', error);
        }
    };

    const clearForm = () => {
        setUrl('');
        setName('');
        setDir('');
        setTracks([]);
        setProgress({ current: 0, total: 0 });
        setExportPath('');
        setStatus('Ready to start downloading...');
    };

    return (
        <div className="min-h-screen bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 text-white">
            {/* Header */}
            <header className="bg-gray-900/90 backdrop-blur-lg border-b border-gray-700/50 p-6 shadow-lg">
                <div className="max-w-6xl mx-auto flex items-center justify-between">
                    <div className="flex items-center space-x-4">
                        <div className="w-12 h-12 bg-gradient-to-br from-blue-600 to-purple-600 rounded-full flex items-center justify-center shadow-md">
                            <span className="text-2xl">🎵</span>
                        </div>
                        <h1 className="text-3xl font-extrabold bg-gradient-to-r from-blue-400 to-purple-500 bg-clip-text text-transparent">
                            GrooveSync
                        </h1>
                    </div>
                    <p className="text-gray-400 text-sm font-medium">Professional DJ Download Manager</p>
                </div>
            </header>

            <div className="max-w-6xl mx-auto p-8">
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* Main Download Form */}
                    <div className="lg:col-span-2 space-y-8">
                        <div className="bg-gray-800/80 backdrop-blur-lg rounded-2xl p-8 border border-gray-700/50 shadow-xl">
                            <h2 className="text-2xl font-semibold text-white mb-8 flex items-center space-x-3">
                                <span className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center">⚙️</span>
                                <span>Download Settings</span>
                            </h2>

                            <div className="space-y-8">
                                {/* URL Input */}
                                <div>
                                    <label className="block text-sm font-medium text-gray-300 mb-3">
                                        Playlist URL
                                    </label>
                                    <input
                                        type="text"
                                        value={url}
                                        onChange={(e) => setUrl(e.target.value)}
                                        placeholder="Paste YouTube, SoundCloud, or Spotify playlist URL..."
                                        className="w-full p-4 rounded-lg bg-gray-700/70 border border-gray-600/70 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 text-white placeholder-gray-500 transition-all duration-300 hover:border-gray-500"
                                    />
                                </div>

                                {/* Name and Format Row */}
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-3">
                                            Playlist Name
                                        </label>
                                        <input
                                            type="text"
                                            value={name}
                                            onChange={(e) => setName(e.target.value)}
                                            placeholder="My Awesome Playlist"
                                            className="w-full p-4 rounded-lg bg-gray-700/70 border border-gray-600/70 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 text-white placeholder-gray-500 transition-all duration-300 hover:border-gray-500"
                                        />
                                    </div>

                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-3">
                                            Audio Format
                                        </label>
                                        <select
                                            value={format}
                                            onChange={(e) => setFormat(e.target.value)}
                                            className="w-full p-4 rounded-lg bg-gray-700/70 border border-gray-600/70 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 text-white transition-all duration-300 hover:border-gray-500"
                                        >
                                            <option value="wav">WAV (Highest Quality)</option>
                                            <option value="mp3">MP3 (Standard)</option>
                                            <option value="aac">AAC (Compressed)</option>
                                            <option value="flac">FLAC (Lossless)</option>
                                        </select>
                                    </div>
                                </div>

                                {/* Directory Input */}
                                <div>
                                    <label className="block text-sm font-medium text-gray-300 mb-3">
                                        Download Directory (Optional)
                                    </label>
                                    <input
                                        type="text"
                                        value={dir}
                                        onChange={(e) => setDir(e.target.value)}
                                        placeholder="./downloads (default)"
                                        className="w-full p-4 rounded-lg bg-gray-700/70 border border-gray-600/70 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 text-white placeholder-gray-500 transition-all duration-300 hover:border-gray-500"
                                    />
                                </div>

                                {/* Action Buttons */}
                                <div className="flex flex-wrap gap-4 pt-6">
                                    <button
                                        onClick={handleStartDownload}
                                        disabled={isDownloading || !url || !name}
                                        className={`flex-1 py-4 px-6 rounded-lg font-semibold transition-all duration-300 shadow-md ${
                                            isDownloading || !url || !name
                                                ? 'bg-gray-600 cursor-not-allowed text-gray-400'
                                                : 'bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white hover:shadow-xl hover:scale-105'
                                        }`}
                                    >
                                        {isDownloading ? (
                                            <span className="flex items-center justify-center">
                                                <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin mr-3"></div>
                                                Downloading...
                                            </span>
                                        ) : (
                                            'Start Download'
                                        )}
                                    </button>

                                    <button
                                        onClick={handleExportToRekordbox}
                                        disabled={isDownloading || !name || progress.current < progress.total || progress.total === 0}
                                        className={`flex-1 py-4 px-6 rounded-lg font-semibold transition-all duration-300 shadow-md ${
                                            isDownloading || !name || progress.current < progress.total || progress.total === 0
                                                ? 'bg-gray-600 cursor-not-allowed text-gray-400'
                                                : 'bg-gradient-to-r from-green-600 to-teal-600 hover:from-green-700 hover:to-teal-700 text-white hover:shadow-xl hover:scale-105'
                                        }`}
                                    >
                                        Export to Rekordbox
                                    </button>

                                    <button
                                        onClick={clearForm}
                                        className="px-8 py-4 rounded-lg font-semibold border border-gray-600/70 hover:bg-gray-700/70 text-white transition-all duration-300 shadow-md hover:shadow-lg hover:scale-105"
                                    >
                                        Clear
                                    </button>
                                </div>
                            </div>
                        </div>

                        {/* Progress Panel */}
                        {progress.total > 0 && (
                            <div className="bg-gray-800/80 backdrop-blur-lg rounded-2xl p-6 border border-gray-700/50 shadow-xl">
                                <h3 className="text-lg font-semibold mb-6 text-white flex items-center space-x-3">
                                    <span className="w-8 h-8 bg-purple-600 rounded-full flex items-center justify-center">📊</span>
                                    <span>Download Progress ({progress.current} / {progress.total} tracks)</span>
                                </h3>
                                <div className="relative w-full bg-gray-700/50 rounded-full h-6 overflow-hidden">
                                    <div
                                        className="absolute inset-0 bg-gradient-to-r from-blue-500 to-purple-500 h-6 rounded-full transition-all duration-500 flex items-center justify-center"
                                        style={{ width: `${Math.max((progress.current / progress.total) * 100, 5)}%` }}
                                    >
                                        <span className="text-xs font-bold text-white drop-shadow-md">
                                            {Math.round((progress.current / progress.total) * 100)}%
                                        </span>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Sidebar */}
                    <div className="space-y-8">
                        {/* Status Panel */}
                        <div className="bg-gray-800/80 backdrop-blur-lg rounded-2xl p-6 border border-gray-700/50 shadow-xl">
                            <h3 className="text-lg font-semibold mb-6 text-white flex items-center space-x-3">
                                <span className="w-8 h-8 bg-yellow-600 rounded-full flex items-center justify-center">ℹ️</span>
                                <span>Status</span>
                            </h3>
                            <div
                                className={`p-4 rounded-lg break-words transition-all duration-300 ${
                                    isDownloading
                                        ? 'bg-yellow-500/20 text-yellow-300 border border-yellow-500/40'
                                        : status.includes('successfully')
                                        ? 'bg-green-500/20 text-green-300 border border-green-500/40'
                                        : status.includes('Error') || status.includes('failed')
                                        ? 'bg-red-500/20 text-red-300 border border-red-500/40'
                                        : 'bg-gray-700/70 text-gray-300 border border-gray-600/40'
                                }`}
                            >
                                {status}
                            </div>
                            {exportPath && (
                                <div className="mt-4 p-4 bg-green-500/10 rounded-lg border border-green-500/30">
                                    <p className="text-sm text-green-300">
                                        Exported File: <a href={exportPath} className="underline hover:text-green-200">{exportPath}</a>
                                    </p>
                                </div>
                            )}
                        </div>

                        {/* Track List */}
                        {tracks.length > 0 && (
                            <div className="bg-gray-800/80 backdrop-blur-lg rounded-2xl p-6 border border-gray-700/50 shadow-xl">
                                <h3 className="text-lg font-semibold mb-6 text-white flex items-center space-x-3">
                                    <span className="w-8 h-8 bg-teal-600 rounded-full flex items-center justify-center">🎧</span>
                                    <span>Tracks ({tracks.length})</span>
                                </h3>
                                <div className="max-h-80 overflow-y-auto space-y-3 pr-2 custom-scrollbar">
                                    {tracks.map((track, index) => (
                                        <div
                                            key={index}
                                            className={`flex items-center space-x-4 p-4 rounded-lg transition-all duration-300 ${
                                                index < progress.current
                                                    ? 'bg-green-500/20 text-green-300 border border-green-500/40'
                                                    : index === progress.current && isDownloading
                                                    ? 'bg-yellow-500/20 text-yellow-300 border border-yellow-500/40 animate-pulse'
                                                    : 'bg-gray-700/70 text-gray-300 border border-gray-600/40'
                                            }`}
                                        >
                                            <span className="text-blue-400 font-mono text-sm font-bold min-w-[2.5rem]">
                                                {String(index + 1).padStart(2, '0')}.
                                            </span>
                                            <span className="text-sm flex-1 truncate">{track.title}</span>
                                            {index < progress.current && (
                                                <span className="text-green-400 min-w-[1rem]">✓</span>
                                            )}
                                            {index === progress.current && isDownloading && (
                                                <span className="text-yellow-400 min-w-[1rem] animate-bounce">⬇</span>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {/* Recent Downloads */}
                        {downloadHistory.length > 0 && (
                            <div className="bg-gray-800/80 backdrop-blur-lg rounded-2xl p-6 border border-gray-700/50 shadow-xl">
                                <h3 className="text-lg font-semibold mb-6 text-white flex items-center space-x-3">
                                    <span className="w-8 h-8 bg-green-600 rounded-full flex items-center justify-center">📜</span>
                                    <span>Recent Downloads</span>
                                </h3>
                                <div className="space-y-3">
                                    {downloadHistory.map((item, index) => (
                                        <div
                                            key={index}
                                            className="flex items-center space-x-4 p-4 bg-green-500/10 rounded-lg border border-green-500/30 transition-all duration-300 hover:bg-green-500/20"
                                        >
                                            <span className="text-green-400 min-w-[1.5rem]">✅</span>
                                            <span className="text-sm text-gray-300 flex-1">{item}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </div>

            {/* Footer */}
            <footer className="bg-gray-900/90 backdrop-blur-lg border-t border-gray-700/50 p-4 mt-8">
                <div className="max-w-6xl mx-auto text-center text-gray-400 text-sm">
                    <p>© 2025 GrooveSync. All rights reserved.</p>
                    <p>Built with ❤️ for DJs worldwide.</p>
                </div>
            </footer>
        </div>
    );
}

export default App;