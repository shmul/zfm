import pydub
import pydub.silence
from typing import List, Tuple, Dict, Optional


def analyze_tail_silence(audio: pydub.AudioSegment, 
                        analysis_duration: float = 30.0,
                        silence_thresh: float = -40.0,
                        min_silence_len: float = 1.0) -> Dict:
    """
    Analyze the tail of an audio segment for silence detection.
    
    Args:
        audio: AudioSegment to analyze
        analysis_duration: Duration in seconds to analyze from the end
        silence_thresh: Threshold in dBFS below which is considered silence
        min_silence_len: Minimum duration in seconds for silence detection
        
    Returns:
        dict: Analysis results including silence segments and suggested trim point
    """
    total_len_ms = len(audio)
    analysis_len_ms = int(analysis_duration * 1000)
    
    # Analyze last N seconds of audio
    tail_start_ms = max(0, total_len_ms - analysis_len_ms)
    tail_segment = audio[tail_start_ms:]
    
    # Get volume statistics
    tail_dbfs = tail_segment.dBFS if tail_segment else float('-inf')
    tail_max_dbfs = tail_segment.max_dBFS if tail_segment else float('-inf')
    
    # Detect silence segments
    silence_segments = pydub.silence.detect_silence(
        tail_segment,
        min_silence_len=int(min_silence_len * 1000),
        silence_thresh=silence_thresh
    )
    
    # Calculate suggested trim point
    suggested_trim_start_ms = calculate_trim_point(silence_segments, tail_start_ms, total_len_ms)
    suggested_trim_from_end_ms = (total_len_ms - suggested_trim_start_ms) if suggested_trim_start_ms else None
    
    return {
        'total_duration_ms': total_len_ms,
        'analysis_start_ms': tail_start_ms,
        'tail_dbfs': tail_dbfs,
        'tail_max_dbfs': tail_max_dbfs,
        'silence_segments': silence_segments,
        'suggested_trim_start_ms': suggested_trim_start_ms,
        'suggested_trim_from_end_ms': suggested_trim_from_end_ms,
        'suggested_trim_from_end_seconds': suggested_trim_from_end_ms / 1000.0 if suggested_trim_from_end_ms else None
    }


def calculate_trim_point(silence_segments: List[Tuple[int, int]], 
                        tail_start_ms: int, 
                        total_len_ms: int) -> Optional[int]:
    """
    Calculate suggested trim point based on silence segments.
    
    Args:
        silence_segments: List of (start, end) tuples in milliseconds
        tail_start_ms: Start of tail analysis window
        total_len_ms: Total audio length in milliseconds
        
    Returns:
        Suggested trim point in milliseconds from start of audio, or None
    """
    if not silence_segments:
        return None
    
    # Find the earliest silence that extends to the end
    for start_ms, end_ms in silence_segments:
        # Convert relative positions to absolute positions
        abs_start_ms = tail_start_ms + start_ms
        abs_end_ms = tail_start_ms + end_ms
        
        # If silence extends to the very end, suggest trimming at silence start
        if abs_end_ms >= total_len_ms - 100:  # 100ms tolerance for end detection
            return abs_start_ms
    
    return None


def get_volume_profile(audio: pydub.AudioSegment, 
                      window_size: float = 1.0) -> List[Tuple[float, float]]:
    """
    Get volume profile of audio segment in windows.
    
    Args:
        audio: AudioSegment to analyze
        window_size: Size of analysis window in seconds
        
    Returns:
        List of (timestamp, dBFS) tuples
    """
    window_ms = int(window_size * 1000)
    profile = []
    
    for i in range(0, len(audio), window_ms):
        window = audio[i:i + window_ms]
        if len(window) > 0:
            timestamp = i / 1000.0
            dbfs = window.dBFS
            profile.append((timestamp, dbfs))
    
    return profile


def format_silence_analysis(analysis: Dict) -> str:
    """
    Format silence analysis results for display.
    
    Args:
        analysis: Analysis results from analyze_tail_silence
        
    Returns:
        Formatted string for display
    """
    lines = []
    lines.append(f"Audio duration: {analysis['total_duration_ms'] / 1000:.1f}s")
    lines.append(f"Tail volume: {analysis['tail_dbfs']:.1f} dBFS (peak: {analysis['tail_max_dbfs']:.1f} dBFS)")
    
    if analysis['silence_segments']:
        lines.append(f"Found {len(analysis['silence_segments'])} silence segments in tail:")
        for i, (start, end) in enumerate(analysis['silence_segments']):
            duration = (end - start) / 1000.0
            abs_start = (analysis['analysis_start_ms'] + start) / 1000.0
            lines.append(f"  {i+1}. {abs_start:.1f}s - {abs_start + duration:.1f}s ({duration:.1f}s)")
    else:
        lines.append("No silence segments found in tail")
    
    if analysis['suggested_trim_from_end_seconds']:
        lines.append(f"Suggested trim: {analysis['suggested_trim_from_end_seconds']:.1f}s from end")
    
    return "\n".join(lines)