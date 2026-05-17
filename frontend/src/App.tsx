import { useState, useEffect } from 'react';
import { Search as SearchIcon, Settings2, FileText, Folder, HardDrive, EyeOff, Download, Activity, Check, Sparkles, Zap, Brain } from 'lucide-react';
import { DownloadLLM, GetSettings, Search, AISearch, OpenFile, OpenFolder } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

export default function App() {
  const [activeTab, setActiveTab] = useState<'search' | 'settings'>('search');
  
  const [downloadProgress, setDownloadProgress] = useState(0);
  const [isDownloading, setIsDownloading] = useState(true);
  const [isDownloaded, setIsDownloaded] = useState(false);

  useEffect(() => {
    EventsOn("llm-download-progress", (data: any) => {
      setIsDownloading(true);
      setDownloadProgress(data.Percentage);
    });
    EventsOn("llm-download-complete", () => {
      setIsDownloading(false);
      setIsDownloaded(true);
      setDownloadProgress(100);
    });
    EventsOn("llm-download-error", (err: any) => {
      console.error(err);
      setIsDownloading(false);
    });
    DownloadLLM().catch(console.error);
  }, []);

  return (
    <div className="w-screen h-screen bg-[#0a0a0f] text-zinc-100 flex flex-col overflow-hidden selection:bg-indigo-500/20">
      {/* Ambient Background Gradient */}
      <div className="fixed inset-0 pointer-events-none">
        <div className="absolute top-0 left-1/4 w-96 h-96 bg-indigo-600/5 rounded-full blur-[120px]" />
        <div className="absolute bottom-0 right-1/4 w-96 h-96 bg-emerald-600/5 rounded-full blur-[120px]" />
      </div>
      
      {/* Header */}
      <div className="h-12 flex items-center justify-between px-5 border-b border-white/[0.06] bg-[#0a0a0f]/90 backdrop-blur-xl z-10 relative" style={{ WebkitAppRegion: 'drag' } as any}>
        <div className="flex items-center gap-1.5">
          <div className="w-3 h-3 rounded-full bg-white/[0.06] hover:bg-red-500/80 transition-colors" />
          <div className="w-3 h-3 rounded-full bg-white/[0.06] hover:bg-yellow-500/80 transition-colors" />
          <div className="w-3 h-3 rounded-full bg-white/[0.06] hover:bg-green-500/80 transition-colors" />
        </div>

        <div className="absolute left-1/2 -translate-x-1/2 flex items-center gap-2 text-zinc-400 text-xs font-medium tracking-wide">
          <img src="/images/logo.png" alt="Neuron" className="w-5 h-5 rounded" />
          <span>Neuron Search</span>
        </div>
        
        <div className="flex gap-0.5" style={{ WebkitAppRegion: 'no-drag' } as any}>
          <button 
            onClick={() => setActiveTab('search')}
            className={`px-3 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 ${activeTab === 'search' ? 'bg-white/[0.08] text-zinc-100 shadow-sm' : 'text-zinc-500 hover:text-zinc-300'}`}
          >
            <SearchIcon className="w-3.5 h-3.5" />
            Search
          </button>
          <button 
            onClick={() => setActiveTab('settings')}
            className={`px-3 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 ${activeTab === 'settings' ? 'bg-white/[0.08] text-zinc-100 shadow-sm' : 'text-zinc-500 hover:text-zinc-300'}`}
          >
            <Settings2 className="w-3.5 h-3.5" />
            Settings
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden relative">
        {activeTab === 'search' ? <SearchView /> : <SettingsView isDownloading={isDownloading} isDownloaded={isDownloaded} downloadProgress={downloadProgress} />}
      </div>
    </div>
  );
}

function SearchView() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<any[]>([]);
  const [aiResults, setAiResults] = useState<any[]>([]);
  const [phase2Loading, setPhase2Loading] = useState(false);
  const [isIndexing, setIsIndexing] = useState(false);

  useEffect(() => {
    EventsOn("index-start", () => setIsIndexing(true));
    EventsOn("index-complete", () => setIsIndexing(false));
  }, []);

  useEffect(() => {
    if (query.trim().length > 0) {
      Search(query).then((res) => {
        setResults(res || []);
      }).catch(console.error);

      if (query.length > 2) {
        setPhase2Loading(true);
        const timer = setTimeout(() => {
          AISearch(query).then((res) => {
            setAiResults(res || []);
            setPhase2Loading(false);
          }).catch(() => setPhase2Loading(false));
        }, 800);
        return () => clearTimeout(timer);
      }
    } else {
      setResults([]);
      setAiResults([]);
      setPhase2Loading(false);
    }
  }, [query]);

  return (
    <div className="h-full flex flex-col items-center pt-12 px-6 relative w-full">
      {/* Search Bar */}
      <div className="w-full max-w-3xl relative group flex-shrink-0 search-glow rounded-2xl transition-all duration-300">
        <div className="absolute inset-y-0 left-0 flex items-center pl-5 pointer-events-none">
          <SearchIcon className="w-5 h-5 text-zinc-500 group-focus-within:text-indigo-400 transition-colors duration-200" />
        </div>
        <input 
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="w-full bg-white/[0.04] border-0 rounded-2xl py-4 pl-14 pr-14 text-base text-zinc-100 placeholder-zinc-600 focus:outline-none transition-all"
          placeholder="Search your files with AI..."
          autoFocus
        />
        
        <div className="absolute inset-y-0 right-0 flex items-center pr-5 gap-2">
          {phase2Loading && (
            <div className="w-4 h-4 border-2 border-indigo-500/40 border-t-indigo-400 rounded-full animate-spin" />
          )}
          {isIndexing && !phase2Loading && (
            <div className="flex items-center gap-1.5 text-emerald-500/70">
              <div className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
              <span className="text-[10px] font-medium uppercase tracking-wider">Indexing</span>
            </div>
          )}
        </div>
      </div>

      {/* Results */}
      <div className="w-full max-w-6xl mt-6 flex-1 overflow-hidden">
        {query.length > 0 ? (
          <div className="h-full grid grid-cols-2 gap-6">
            
            {/* Left: Instant Results */}
            <div className="flex flex-col h-full overflow-hidden">
              <div className="text-[11px] font-semibold text-zinc-500 tracking-[0.15em] uppercase mb-3 flex items-center gap-2 shrink-0">
                <Zap className="w-3 h-3 text-amber-500" />
                Instant Matches
                <span className="ml-auto text-zinc-600 font-normal tracking-normal normal-case">{results.length} found</span>
              </div>
              <div className="flex-1 overflow-y-auto scrollbar-hide space-y-2 pr-1">
                {results.length > 0 ? (
                  results.map((r, i) => (
                    <div key={i} className="fade-up" style={{ animationDelay: `${i * 40}ms` }}>
                      <ResultItem 
                        title={r.filename} 
                        path={r.path} 
                        tag={r.tags?.[0] || 'File'} 
                        icon={<FileText className="w-4 h-4" />} 
                        match={r.summary?.length > 120 ? r.summary.substring(0, 120) + "…" : r.summary} 
                        onClick={() => OpenFile(r.path)}
                        onOpenFolder={(e: any) => { e.stopPropagation(); OpenFolder(r.path); }}
                      />
                    </div>
                  ))
                ) : (
                  <div className="flex flex-col items-center justify-center h-32 text-zinc-600">
                    <SearchIcon className="w-8 h-8 mb-2 opacity-20" />
                    <p className="text-sm">No instant matches</p>
                  </div>
                )}
              </div>
            </div>

            {/* Right: AI Results */}
            <div className="flex flex-col h-full overflow-hidden border-l border-white/[0.04] pl-6">
              <div className="text-[11px] font-semibold text-emerald-400/80 tracking-[0.15em] uppercase mb-3 flex items-center gap-2 shrink-0">
                <Sparkles className="w-3 h-3" />
                AI Deep Search
                {aiResults.length > 0 && <span className="ml-auto text-emerald-500/50 font-normal tracking-normal normal-case">{aiResults.length} curated</span>}
              </div>
              <div className="flex-1 overflow-y-auto scrollbar-hide space-y-2 pr-1 relative">
                {phase2Loading ? (
                   <div className="flex flex-col items-center justify-center h-40 space-y-3">
                     <div className="relative">
                       <Brain className="w-10 h-10 text-emerald-500/30" />
                       <div className="absolute inset-0 flex items-center justify-center">
                         <div className="w-4 h-4 border-2 border-emerald-500/40 border-t-emerald-400 rounded-full animate-spin" />
                       </div>
                     </div>
                     <p className="text-xs text-zinc-500">AI is analyzing documents…</p>
                   </div>
                ) : query.length > 2 && aiResults.length > 0 ? (
                  <div className="space-y-2">
                    {aiResults.map((r, i) => (
                      <div key={`ai-${i}`} className="fade-up" style={{ animationDelay: `${i * 60}ms` }}>
                        <ResultItem 
                          title={r.filename} 
                          path={r.path} 
                          tag="AI Match"
                          variant="ai"
                          icon={<Sparkles className="w-4 h-4 text-emerald-400" />} 
                          aiMatch={r.aiReason || 'Semantically relevant'}
                          onClick={() => OpenFile(r.path)}
                          onOpenFolder={(e: any) => { e.stopPropagation(); OpenFolder(r.path); }}
                        />
                      </div>
                    ))}
                  </div>
                ) : query.length > 2 ? (
                  <div className="flex flex-col items-center justify-center h-32 text-zinc-600">
                    <Sparkles className="w-8 h-8 mb-2 opacity-20" />
                    <p className="text-sm">No relevant AI matches</p>
                  </div>
                ) : null}
              </div>
            </div>

          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-zinc-600 pb-20">
            <div className="relative mb-6">
              <SearchIcon className="w-20 h-20 opacity-[0.07]" />
              <div className="absolute bottom-0 right-0 w-8 h-8 rounded-full bg-indigo-500/10 flex items-center justify-center">
                <Sparkles className="w-4 h-4 text-indigo-400/50" />
              </div>
            </div>
            <p className="text-lg font-light text-zinc-500">Search your files with AI</p>
            <p className="text-xs text-zinc-700 mt-2">Type to search instantly · AI reranks results automatically</p>
            {isIndexing && (
              <div className="mt-6 flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/5 border border-emerald-500/10">
                <div className="w-1.5 h-1.5 rounded-full bg-emerald-500 pulse-ring" />
                <span className="text-[10px] text-emerald-500/70 font-medium uppercase tracking-wider">Indexing your files</span>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function ResultItem({ title, path, tag, icon, match, aiMatch, onClick, onOpenFolder, variant }: any) {
  const isAi = variant === 'ai';
  return (
    <div 
      onClick={onClick}
      className={`result-card flex items-start gap-3 p-3.5 rounded-xl cursor-pointer group relative
        ${isAi 
          ? 'bg-emerald-500/[0.04] border border-emerald-500/[0.08] hover:bg-emerald-500/[0.08] hover:border-emerald-500/[0.15]' 
          : 'bg-white/[0.02] border border-white/[0.05] hover:bg-white/[0.05] hover:border-white/[0.1]'
        }`}
    >
      <div className={`p-2 rounded-lg transition-colors shrink-0 ${isAi ? 'bg-emerald-500/10 text-emerald-400' : 'bg-white/[0.04] text-zinc-500 group-hover:text-zinc-300'}`}>
        {icon}
      </div>
      <div className="flex-1 min-w-0 pr-7">
        <div className="flex items-center gap-2 mb-1">
          <h3 className="font-medium text-[13px] text-zinc-200 truncate">{title}</h3>
          <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium shrink-0 
            ${isAi ? 'bg-emerald-500/10 text-emerald-400/80' : 'bg-white/[0.04] text-zinc-500'}`}>
            {tag}
          </span>
        </div>
        <p className="text-[11px] text-zinc-600 flex items-center gap-1 truncate mb-1">
          <Folder className="w-3 h-3 shrink-0" />
          <span className="truncate">{path}</span>
        </p>
        {match && (
          <p className="text-[11px] text-zinc-500 mt-1 line-clamp-2 leading-relaxed">{match}</p>
        )}
        {aiMatch && (
          <p className="text-[11px] text-emerald-400/70 mt-1.5 flex items-center gap-1">
            <Sparkles className="w-3 h-3 shrink-0" />
            <span className="line-clamp-1">{aiMatch}</span>
          </p>
        )}
      </div>
      
      {onOpenFolder && (
        <button 
          onClick={onOpenFolder}
          title="Show in Finder"
          className="absolute right-3 top-3 p-1.5 rounded-lg text-zinc-600 hover:text-zinc-200 hover:bg-white/[0.06] opacity-0 group-hover:opacity-100 transition-all"
        >
          <Folder className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  );
}

function SettingsView({ isDownloading, isDownloaded, downloadProgress }: any) {
  return (
    <div className="h-full flex flex-col p-8 w-full max-w-3xl mx-auto overflow-y-auto scrollbar-hide">
      <h1 className="text-xl font-semibold text-zinc-100 mb-8 flex items-center gap-3">
        <Settings2 className="w-5 h-5 text-zinc-500" />
        Preferences
      </h1>
      
      <div className="space-y-6">
        {/* Directories */}
        <section className="space-y-3">
          <h2 className="text-[11px] font-semibold text-zinc-500 uppercase tracking-[0.15em]">Indexed Directories</h2>
          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl overflow-hidden">
            <div className="p-4 border-b border-white/[0.04] flex justify-between items-center">
              <div className="flex items-center gap-3 text-zinc-300 text-sm">
                <HardDrive className="w-4 h-4 text-zinc-500" />
                <span>~/Documents</span>
              </div>
              <button className="text-[11px] text-red-400/60 hover:text-red-400 transition-colors">Remove</button>
            </div>
            <div className="p-3 hover:bg-white/[0.02] transition-colors cursor-pointer text-center">
              <button className="text-xs text-zinc-500 hover:text-zinc-300 font-medium transition-colors">+ Add Directory</button>
            </div>
          </div>
        </section>

        {/* Ignore Rules */}
        <section className="space-y-3">
          <h2 className="text-[11px] font-semibold text-zinc-500 uppercase tracking-[0.15em]">Ignore Rules</h2>
          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
            <div className="flex items-start gap-4">
              <EyeOff className="w-4 h-4 text-zinc-600 mt-0.5 shrink-0" />
              <div className="flex-1">
                <label className="text-sm font-medium text-zinc-300 mb-1 block">Custom Ignore Prompts</label>
                <p className="text-[11px] text-zinc-600 mb-3">Tell the AI what to skip during indexing.</p>
                <textarea 
                  className="w-full bg-black/20 border border-white/[0.06] rounded-lg p-3 text-sm text-zinc-300 focus:outline-none focus:border-indigo-500/30 resize-none h-20 transition-colors"
                  defaultValue="Ignore node_modules, build artifacts, and personal banking documents."
                  placeholder="E.g. Skip financial records..."
                />
              </div>
            </div>
          </div>
        </section>
        
        {/* AI Engine */}
        <section className="space-y-3">
          <h2 className="text-[11px] font-semibold text-zinc-500 uppercase tracking-[0.15em]">AI Engine</h2>
          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium text-sm text-zinc-200 flex items-center gap-2">
                  <Brain className="w-4 h-4 text-indigo-400" />
                  Local SLM
                  {isDownloaded && <span className="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 text-[10px] font-semibold">Ready</span>}
                </h3>
                <p className="text-[11px] text-zinc-600 mt-1 ml-6">TinyLlama 1.1B · Q4 Quantized · 630MB</p>
              </div>
              
              {!isDownloaded && !isDownloading && (
                <span className="text-xs text-zinc-600">Waiting…</span>
              )}
              
              {isDownloading && (
                <div className="flex items-center gap-2">
                  <Activity className="w-3.5 h-3.5 text-indigo-400 animate-pulse" />
                  <span className="text-xs font-medium text-indigo-400">{downloadProgress}%</span>
                </div>
              )}
              
              {isDownloaded && (
                <Check className="w-4 h-4 text-emerald-500" />
              )}
            </div>

            {isDownloading && (
              <div className="w-full h-1 bg-white/[0.04] rounded-full overflow-hidden">
                <div 
                  className="h-full bg-gradient-to-r from-indigo-500 to-emerald-500 rounded-full transition-all duration-500 ease-out" 
                  style={{ width: `${downloadProgress}%` }}
                />
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
