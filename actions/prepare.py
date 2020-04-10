import pydub
import typing
import pytimeparse
import pathlib
import os

Msecs = typing.NewType('Msecs', int)
AudioFiles = typing.NewType('AudioFiles', typing.List[str])


def offset(hms: str, sec: float) -> Msecs:
    s = None
    if hms:
        s = pytimeparse.parse(hms)
    if s != None:
        return Msecs(s * 1000)
    return Msecs(sec * 1000)


def prepare(file: str = '',
            start: str = '',
            end: str = '',
            head: float = 0,
            tail: float = 0,
            fade_in: float = 0,
            fade_out: float = 0) -> pydub.AudioSegment:
    file = os.path.realpath(file)
    audio = pydub.AudioSegment.from_file(pathlib.Path(file))
    ln = len(audio)
    if tail == 0:
        tail = ln / 1000
        s = offset(start, head)
        e = offset(end, tail)
        print("file {}, len: {}, start: {}, end: {}".format(file, ln, s, e))
        audio = audio[s:e]
    if fade_in:
        audio = audio.fade_in(int(fade_in * 1000))
    if fade_out:
        audio = audio.fade_out(int(fade_out * 1000))

    return audio
