import { useState, useEffect, useRef } from 'react';
import { StartDownload, ExportToRekordbox, GetAllDownloads } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { Music, Link, Folder, Download, FileText, RotateCcw, Palette, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import WaveSurfer from 'wavesurfer.js';
import { motion, AnimatePresence } from 'framer-motion';
import './App.css';
import { main } from '../wailsjs/go/models'; // Import the main namespace

interface Progress {
    current: number;
    total: number;
}

interface Track {
    url: string;
    title: string;
    audioPath?: string;
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
    const [exportPath, setExportPath] = useState<string>('');
    const [theme, setTheme] = useState<'dark' | 'neon'>('neon');
    const waveformRefs = useRef<Map<number, WaveSurfer>>(new Map());
    const [jobs, setJobs] = useState<main.DownloadJob[]>([]); // Jobs state for real-time updates

    useEffect(() => {
        const savedTheme = localStorage.getItem('theme') as 'dark' | 'neon' || 'neon';
        setTheme(savedTheme);
        document.documentElement.classList.toggle('neon-theme', savedTheme === 'neon');
        document.documentElement.classList.toggle('dark-theme', savedTheme === 'dark');

        // Backend event listeners (added 'settingsUpdated' and 'downloadStatsUpdated' to fix "No listeners" logs)
        EventsOn('downloadCreated', (job: main.DownloadJob) => {
            setJobs(prev => [...prev, main.DownloadJob.createFrom(job)]);
            setStatus('Download job created. Extracting tracks...');
        });
        EventsOn('downloadUpdated', (job: main.DownloadJob) => {
            setJobs(prev => prev.map(j => j.id === job.id ? main.DownloadJob.createFrom(job) : j));
            setStatus(job.status);
            if (job.status === "completed") {
                setIsDownloading(false);
                setDownloadHistory(prev => [`${job.title} - ${new Date().toLocaleTimeString()}`, ...prev.slice(0, 4)]);
            } else if (job.status === "failed") {
                setIsDownloading(false);
                setStatus(job.error || 'Download failed. Check logs.');
            }
            setProgress({ current: Math.round(job.progress), total: 100 });
        });
        EventsOn('tracks', (tracks: Track[]) => {
            setTracks(tracks);
        });
        EventsOn('exportCompleted', (data: { path: string }) => {
            setExportPath(data.path);
            setStatus(`Exported to Rekordbox successfully! File: ${data.path}`);
        });
        EventsOn('settingsUpdated', (settings: main.AppSettings) => {
            console.log('Settings updated:', settings); // Optional: Update UI if needed (e.g., theme)
        });
        EventsOn('downloadStatsUpdated', (stats: { total: number, active: number, completed: number }) => {
            console.log('Download stats:', stats); // Optional: Update UI dashboard if added
        });

        GetAllDownloads().then(jobs => setJobs(jobs.map(j => main.DownloadJob.createFrom(j))));

        return () => {
            waveformRefs.current.forEach(ws => ws.destroy());
        };
    }, []);

    useEffect(() => {
        tracks.forEach((track, index) => {
            if (index < Math.round(progress.current / 100 * tracks.length) && !waveformRefs.current.has(index)) {
                const container = document.getElementById(`waveform-${index}`);
                if (container && jobs.find(j => j.status === "completed")) {
                    const audioPath = `${jobs.find(j => j.status === "completed")?.outputPath}/${track.title}.${format}`;
                    const ws = WaveSurfer.create({
                        container,
                        waveColor: '#00FFFF',
                        progressColor: '#9B5DE5',
                        height: 40,
                        barWidth: 2,
                        barGap: 1,
                    });
                    ws.load(audioPath);
                    waveformRefs.current.set(index, ws);
                }
            }
        });
    }, [tracks, progress.current, jobs, format]);

    const handleStartDownload = async () => {
        if (!url || !name) {
            setStatus('Error: Playlist URL and Name are required');
            return;
        }
        setIsDownloading(true);
        setProgress({ current: 0, total: 0 });
        setExportPath('');
        setTracks([]);
        try {
            await StartDownload(url, name, format, dir);
            setStatus('Download initiated. Check progress...');
        } catch (error) {
            setStatus('Download initiation failed. Check logs.');
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
        setJobs([]);
    };

    const toggleTheme = () => {
        const newTheme = theme === 'neon' ? 'dark' : 'neon';
        setTheme(newTheme);
        localStorage.setItem('theme', newTheme);
        document.documentElement.classList.toggle('neon-theme', newTheme === 'neon');
        document.documentElement.classList.toggle('dark-theme', newTheme === 'dark');
    };

    return (
        <div className="min-h-screen bg-club-gradient text-white flex flex-col">
            <header className="bg-navy-dark/90 backdrop-blur-lg border-b border-club-gray p-4 shadow-neon rounded-t-lg">
                <div className="max-w-7xl mx-auto flex items-center justify-between">
                    <div className="flex items-center space-x-4">
                        <motion.div initial={{ scale: 0 }} animate={{ scale: 1 }} transition={{ duration: 0.5 }}>
                            <Music className="w-10 h-10 text-neon-blue" />
                        </motion.div>
                        <h1 className="text-2xl font-extrabold text-neon-blue">GrooveSync</h1>
                    </div>
                    <p className="text-text-secondary text-xs font-medium">Professional DJ Download Manager</p>
                    <button onClick={toggleTheme} className="p-1 rounded-full hover:bg-club-gray">
                        <Palette className="w-4 h-4 text-neon-purple" />
                    </button>
                </div>
            </header>

            <div className="flex-1 max-w-7xl mx-auto p-6 grid grid-cols-1 lg:grid-cols-4 gap-6">
                <motion.aside
                    className="lg:col-span-1 bg-navy-light/80 backdrop-blur-lg rounded-lg p-4 border border-club-gray shadow-neon space-y-4"
                    initial={{ opacity: 0, x: -50 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ duration: 0.5 }}
                >
                    <h3 className="text-base font-semibold flex items-center space-x-2">
                        <Palette className="w-5 h-5 text-neon-purple" />
                        <span>Settings</span>
                    </h3>
                    <div className="space-y-3">
                        <label className="block text-xs font-medium text-text-secondary">Max Concurrent Downloads</label>
                        <input type="number" defaultValue={4} className="w-full p-1 rounded bg-club-gray border border-club-gray focus:border-neon-blue text-white text-sm" />
                    </div>
                    <div className="space-y-3">
                        <label className="block text-xs font-medium text-text-secondary">Theme</label>
                        <select value={theme} onChange={toggleTheme} className="w-full p-1 rounded bg-club-gray border border-club-gray focus:border-neon-blue text-white text-sm">
                            <option value="neon">Neon (Club Mode)</option>
                            <option value="dark">Dark (Minimal)</option>
                        </select>
                    </div>
                </motion.aside>

                <div className="lg:col-span-3 space-y-6 flex flex-col gap-6">
                    <motion.div
                        className="bg-navy-light/80 backdrop-blur-lg rounded-lg p-6 border border-club-gray shadow-neon flex-1"
                        initial={{ opacity: 0, y: 50 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5 }}
                    >
                        <h2 className="text-xl font-semibold mb-6 flex items-center space-x-2 text-white">
                            <Download className="w-6 h-6 text-neon-blue" />
                            <span>Download Settings</span>
                        </h2>

                        <div className="space-y-4 flex flex-col gap-4">
                            <div className="relative">
                                <label className="block text-xs font-medium text-text-secondary mb-1">Playlist URL</label>
                                <div className="relative">
                                    <Link className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neon-purple" />
                                    <input
                                        type="text"
                                        value={url}
                                        onChange={(e) => setUrl(e.target.value)}
                                        placeholder="Paste YouTube, SoundCloud, or Spotify playlist URL..."
                                        className="w-full pl-9 p-3 rounded-lg bg-club-gray border border-club-gray focus:border-neon-blue focus:ring-neon-blue/30 text-white placeholder-text-secondary transition-all hover:border-neon-blue/50 shadow-neon-hover text-sm"
                                    />
                                </div>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div className="relative">
                                    <label className="block text-xs font-medium text-text-secondary mb-1">Playlist Name</label>
                                    <div className="relative">
                                        <Music className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neon-pink" />
                                        <input
                                            type="text"
                                            value={name}
                                            onChange={(e) => setName(e.target.value)}
                                            placeholder="My Awesome Playlist"
                                            className="w-full pl-9 p-3 rounded-lg bg-club-gray border border-club-gray focus:border-neon-blue focus:ring-neon-blue/30 text-white placeholder-text-secondary transition-all hover:border-neon-blue/50 shadow-neon-hover text-sm"
                                        />
                                    </div>
                                </div>

                                <div className="relative">
                                    <label className="block text-xs font-medium text-text-secondary mb-1">Audio Format</label>
                                    <div className="relative">
                                        <FileText className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neon-blue" />
                                        <select
                                            value={format}
                                            onChange={(e) => setFormat(e.target.value)}
                                            className="w-full pl-9 p-3 rounded-lg bg-club-gray border border-club-gray focus:border-neon-blue focus:ring-neon-blue/30 text-white transition-all hover:border-neon-blue/50 shadow-neon-hover appearance-none text-sm"
                                        >
                                            <option value="wav">WAV (Highest Quality)</option>
                                            <option value="mp3">MP3 (Standard)</option>
                                            <option value="aac">AAC (Compressed)</option>
                                            <option value="flac">FLAC (Lossless)</option>
                                        </select>
                                    </div>
                                </div>
                            </div>

                            <div className="relative">
                                <label className="block text-xs font-medium text-text-secondary mb-1">Download Directory (Optional)</label>
                                <div className="relative">
                                    <Folder className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neon-purple" />
                                    <input
                                        type="text"
                                        value={dir}
                                        onChange={(e) => setDir(e.target.value)}
                                        placeholder="./downloads (default)"
                                        className="w-full pl-9 p-3 rounded-lg bg-club-gray border border-club-gray focus:border-neon-blue focus:ring-neon-blue/30 text-white placeholder-text-secondary transition-all hover:border-neon-blue/50 shadow-neon-hover text-sm"
                                    />
                                </div>
                            </div>

                            <div className="flex flex-row gap-3 pt-3 md:flex-nowrap">
                                <motion.button
                                    onClick={handleStartDownload}
                                    disabled={isDownloading || !url || !name}
                                    className={`flex-1 py-3 px-5 rounded-lg font-semibold transition-all ${isDownloading || !url || !name ? 'bg-club-gray cursor-not-allowed text-text-secondary' : 'bg-neon-blue text-navy-dark hover:bg-neon-blue/80 hover:shadow-neon-hover hover:scale-105'}`}
                                    whileHover={{ scale: 1.05 }}
                                    whileTap={{ scale: 0.95 }}
                                >
                                    {isDownloading ? (
                                        <span className="flex items-center justify-center">
                                            <Loader2 className="w-4 h-4 animate-spin mr-1 text-white" />
                                            Downloading...
                                        </span>
                                    ) : (
                                        'Start Download'
                                    )}
                                </motion.button>

                                <motion.button
                                    onClick={handleExportToRekordbox}
                                    disabled={isDownloading || !name || progress.current < progress.total || progress.total === 0}
                                    className={`flex-1 py-3 px-5 rounded-lg font-semibold transition-all ${isDownloading || !name || progress.current < progress.total || progress.total === 0 ? 'bg-club-gray cursor-not-allowed text-text-secondary' : 'bg-neon-purple text-navy-dark hover:bg-neon-purple/80 hover:shadow-neon-hover hover:scale-105'}`}
                                    whileHover={{ scale: 1.05 }}
                                    whileTap={{ scale: 0.95 }}
                                >
                                    Export to Rekordbox
                                </motion.button>

                                <motion.button
                                    onClick={clearForm}
                                    className="px-6 py-3 rounded-lg font-semibold border border-club-gray hover:bg-club-gray text-white transition-all hover:shadow-neon-hover hover:scale-105"
                                    whileHover={{ scale: 1.05 }}
                                    whileTap={{ scale: 0.95 }}
                                >
                                    <RotateCcw className="inline w-3 h-3 mr-1" />
                                    Clear
                                </motion.button>
                            </div>
                        </div>
                    </motion.div>

                    {progress.total > 0 && (
                        <motion.div
                            className="bg-navy-light/80 backdrop-blur-lg rounded-lg p-4 border border-club-gray shadow-neon"
                            initial={{ opacity: 0, y: 50 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.5, delay: 0.2 }}
                        >
                            <h3 className="text-base font-semibold mb-4 flex items-center space-x-2 text-white">
                                <Download className="w-5 h-5 text-neon-pink" />
                                <span>Download Progress ({progress.current} / {progress.total} tracks)</span>
                            </h3>
                            <div className="relative w-full bg-club-gray rounded-full h-4 overflow-hidden">
                                <motion.div
                                    className="absolute inset-0 bg-gradient-to-r from-neon-blue to-neon-purple h-4 rounded-full flex items-center justify-center"
                                    initial={{ width: 0 }}
                                    animate={{ width: `${(progress.current / progress.total) * 100}%` }}
                                    transition={{ duration: 0.5 }}
                                >
                                    <span className="text-xs font-bold text-white drop-shadow-md">
                                        {Math.round((progress.current / progress.total) * 100)}%
                                    </span>
                                </motion.div>
                            </div>
                        </motion.div>
                    )}

                    {tracks.length > 0 && (
                        <motion.div
                            className="bg-navy-light/80 backdrop-blur-lg rounded-lg p-4 border border-club-gray shadow-neon overflow-hidden"
                            initial={{ opacity: 0, y: 50 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.5, delay: 0.4 }}
                        >
                            <h3 className="text-base font-semibold mb-4 flex items-center space-x-2 text-white">
                                <Music className="w-5 h-5 text-neon-blue" />
                                <span>Tracks ({tracks.length})</span>
                            </h3>
                            <div className="max-h-80 overflow-y-auto space-y-3 pr-2 custom-scrollbar">
                                <AnimatePresence>
                                    {tracks.map((track, index) => (
                                        <motion.div
                                            key={index}
                                            initial={{ opacity: 0, y: 20 }}
                                            animate={{ opacity: 1, y: 0 }}
                                            exit={{ opacity: 0, y: -20 }}
                                            transition={{ duration: 0.3, delay: index * 0.05 }}
                                            className={`flex flex-col p-3 rounded-lg transition-all ${
                                                index < Math.round(progress.current / 100 * tracks.length)
                                                    ? 'bg-neon-blue/20 border-neon-blue/40 text-white'
                                                    : index === Math.round(progress.current / 100 * tracks.length) && isDownloading
                                                    ? 'bg-neon-purple/20 border-neon-purple/40 animate-pulse text-white'
                                                    : 'bg-club-gray border-club-gray text-text-secondary'
                                            }`}
                                        >
                                            <div className="flex items-center space-x-3 mb-1">
                                                <span className="text-neon-pink font-mono text-xs font-bold min-w-[2rem]">
                                                    {String(index + 1).padStart(2, '0')}.
                                                </span>
                                                <span className="text-xs flex-1 truncate">{track.title}</span>
                                                {index < Math.round(progress.current / 100 * tracks.length) ? (
                                                    <CheckCircle className="text-neon-blue min-w-[0.75rem]" />
                                                ) : index === Math.round(progress.current / 100 * tracks.length) && isDownloading ? (
                                                    <Loader2 className="text-neon-purple min-w-[0.75rem] animate-spin" />
                                                ) : null}
                                            </div>
                                            {index < Math.round(progress.current / 100 * tracks.length) && jobs.find(j => j.status === "completed") && (
                                                <div className="waveform-container mt-1">
                                                    <div id={`waveform-${index}`} className="w-full" />
                                                </div>
                                            )}
                                        </motion.div>
                                    ))}
                                </AnimatePresence>
                            </div>
                        </motion.div>
                    )}

                    <motion.div
                        className="bg-navy-light/80 backdrop-blur-lg rounded-lg p-4 border border-club-gray shadow-neon"
                        initial={{ opacity: 0, y: 50 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5, delay: 0.6 }}
                    >
                        <h3 className="text-base font-semibold mb-4 flex items-center space-x-2 text-white">
                            <AlertCircle className="w-5 h-5 text-neon-pink" />
                            <span>Status</span>
                        </h3>
                        <div
                            className={`p-3 rounded-lg break-words transition-all duration-300 ${
                                isDownloading
                                    ? 'bg-neon-purple/20 text-neon-purple border-neon-purple/40'
                                    : status.includes('successfully')
                                    ? 'bg-neon-blue/20 text-neon-blue border-neon-blue/40'
                                    : status.includes('Error') || status.includes('failed')
                                    ? 'bg-neon-pink/20 text-neon-pink border-neon-pink/40'
                                    : 'bg-club-gray text-text-secondary border-club-gray'
                            }`}
                        >
                            {status}
                        </div>
                        {exportPath && (
                            <div className="mt-3 p-3 bg-neon-blue/10 rounded-lg border-neon-blue/30">
                                <p className="text-xs text-neon-blue">
                                    Exported File: <a href={exportPath} className="underline hover:text-neon-blue/80">{exportPath}</a>
                                </p>
                            </div>
                        )}
                    </motion.div>

                    {downloadHistory.length > 0 && (
                        <motion.div
                            className="bg-navy-light/80 backdrop-blur-lg rounded-lg p-4 border border-club-gray shadow-neon"
                            initial={{ opacity: 0, y: 50 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.5, delay: 0.8 }}
                        >
                            <h3 className="text-base font-semibold mb-4 flex items-center space-x-2 text-white">
                                <FileText className="w-5 h-5 text-neon-purple" />
                                <span>Recent Downloads</span>
                            </h3>
                            <div className="space-y-3">
                                {downloadHistory.map((item, index) => (
                                    <div
                                        key={index}
                                        className="flex items-center space-x-3 p-3 bg-neon-blue/10 rounded-lg border-neon-blue/30 transition-all hover:bg-neon-blue/20 hover:shadow-neon-hover"
                                    >
                                        <CheckCircle className="text-neon-blue min-w-[1.25rem]" />
                                        <span className="text-xs text-text-secondary flex-1">{item}</span>
                                    </div>
                                ))}
                            </div>
                        </motion.div>
                    )}
                </div>
            </div>

            <footer className="bg-navy-dark/90 backdrop-blur-lg border-t border-club-gray p-3 mt-6">
                <div className="max-w-7xl mx-auto text-center text-text-secondary text-xs">
                    <p>© 2025 GrooveSync. All rights reserved.</p>
                    <p>Built with ❤️ for DJs worldwide.</p>
                </div>
            </footer>
        </div>
    );
}

export default App;