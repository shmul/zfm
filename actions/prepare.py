import pydub
import typing
import pytimeparse
import pathlib
import os

Msecs = typing.NewType('Msecs', int)
AudioFiles = typing.NewType('AudioFiles', typing.List[str])


def tomsecs(v: float) -> Msecs:
    return Msecs(int(1000 * float(v)))


def offset(hms: str, sec: float) -> Msecs:
    s = None
    if hms:
        s = pytimeparse.parse(hms)
    if s != None:
        return tomsecs(s)
    if not sec:
        return tomsecs(0)

    return tomsecs(sec)


def prepare(file: str = '',
            start: str = '',
            end: str = '',
            head: float = 0,
            tail: float = 0,
            fade_in: float = 0,
            fade_out: float = 0) -> (pydub.AudioSegment, bool):
    print(f"\nDEBUG: Processing file: {file}")
    file = os.path.realpath(file)
    print(f"DEBUG: Resolved path: {file}")
    
    # First convert FLAC to WAV if needed
    file_ext = os.path.splitext(file)[1].lower()
    print(f"DEBUG: File extension: {file_ext}")
    
    try:
        if file_ext == '.flac':
            print("DEBUG: Loading FLAC file...")
            # Try different parameters for FLAC files
            try:
                audio = pydub.AudioSegment.from_file(pathlib.Path(file), format='flac')
            except:
                print("DEBUG: Retrying FLAC load with different parameters...")
                audio = pydub.AudioSegment.from_file(pathlib.Path(file), format='flac', 
                                                   parameters=["-nostdin", "-acodec", "flac"])
        else:
            print(f"DEBUG: Loading file with auto-detected format...")
            audio = pydub.AudioSegment.from_file(pathlib.Path(file))
        
        if audio is None:
            raise ValueError("Audio loaded as None")
            
        print(f"DEBUG: Successfully loaded audio file. Duration: {len(audio)}ms")
        print(f"DEBUG: Audio properties - Channels: {audio.channels}, Frame rate: {audio.frame_rate}, Sample width: {audio.sample_width}")
        
    except Exception as e:
        print(f"ERROR: Failed to load audio file {file}")
        print(f"ERROR details: {str(e)}")
        print(f"ERROR type: {type(e)}")
        return None, False

    try:
        ln = len(audio)
        print(f"DEBUG: Original audio length: {ln}ms")
        
        # Calculate positions
        tl = 0
        if tail is not None:
            tl = -abs(float(tail))
        else:
            tl = ln / 1000
        print(f"DEBUG: Calculated tail position: {tl}")

        s = max(0, offset(start, head))  # Ensure we don't go negative
        e = offset(end, tl)
        if e <= 0 or e > ln:
            e = ln
        print(f"DEBUG: Slice positions - start: {s}ms, end: {e}ms")

        # Verify slice positions
        if s >= ln:
            raise ValueError(f"Start position ({s}ms) exceeds audio length ({ln}ms)")
        if e <= s:
            raise ValueError(f"End position ({e}ms) must be greater than start position ({s}ms)")

        print("DEBUG: Attempting to slice audio...")
        audio = audio[s:e]
        if audio is None:
            raise ValueError("Slicing operation returned None")
        print(f"DEBUG: Slice successful. New length: {len(audio)}ms")

        # Apply effects only if audio segment is valid
        if fade_in:
            print(f"DEBUG: Applying fade in: {fade_in}s")
            fade_ms = tomsecs(fade_in)
            if fade_ms > len(audio):
                print("WARNING: Fade in duration exceeds audio length, adjusting...")
                fade_ms = len(audio)
            audio = audio.fade_in(fade_ms)
        
        if fade_out:
            print(f"DEBUG: Applying fade out: {fade_out}s")
            fade_ms = tomsecs(fade_out)
            if fade_ms > len(audio):
                print("WARNING: Fade out duration exceeds audio length, adjusting...")
                fade_ms = len(audio)
            audio = audio.fade_out(fade_ms)
        
        identical = len(audio) == ln
        print(f"DEBUG: Processing complete. Identical to original: {identical}")
        return audio, identical
        
    except Exception as e:
        print(f"ERROR: Failed during audio processing")
        print(f"ERROR details: {str(e)}")
        print(f"ERROR type: {type(e)}")
        return None, False
