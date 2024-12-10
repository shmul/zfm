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
            audio = pydub.AudioSegment.from_file(pathlib.Path(file), format='flac', parameters=["-nostdin"])
        else:
            print(f"DEBUG: Loading file with auto-detected format...")
            audio = pydub.AudioSegment.from_file(pathlib.Path(file), parameters=["-nostdin"])
        
        print(f"DEBUG: Successfully loaded audio file. Duration: {len(audio)}ms")
        
    except Exception as e:
        print(f"ERROR: Failed to load audio file {file}")
        print(f"ERROR details: {str(e)}")
        print(f"ERROR type: {type(e)}")
        return None, False

    if audio is None:
        print(f"ERROR: Audio loaded as None for file {file}")
        return None, False

    try:
        ln = len(audio)
        print(f"DEBUG: Original audio length: {ln}ms")
        
        tl = 0
        if tail != None:
            tl = -abs(float(tail))
        else:
            tl = ln / 1000
        print(f"DEBUG: Calculated tail position: {tl}")

        s = offset(start, head)
        e = offset(end, tl)
        if e == 0:
            e = ln
        print(f"DEBUG: Slice positions - start: {s}ms, end: {e}ms")

        print("DEBUG: Attempting to slice audio...")
        audio = audio[s:e]
        print(f"DEBUG: Slice successful. New length: {len(audio)}ms")

        if fade_in:
            print(f"DEBUG: Applying fade in: {fade_in}s")
            audio = audio.fade_in(tomsecs(fade_in))
        
        if fade_out:
            print(f"DEBUG: Applying fade out: {fade_out}s")
            audio = audio.fade_out(tomsecs(fade_out))
        
        identical = len(audio) == ln
        print(f"DEBUG: Processing complete. Identical to original: {identical}")
        return audio, identical
        
    except Exception as e:
        print(f"ERROR: Failed during audio processing")
        print(f"ERROR details: {str(e)}")
        print(f"ERROR type: {type(e)}")
        return None, False
