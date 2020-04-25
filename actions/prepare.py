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
    file = os.path.realpath(file)
    audio = pydub.AudioSegment.from_file(pathlib.Path(file),
                                         parameters=["-c", "copy"])
    ln = len(audio)
    if not tail:
        tail = ln / 1000

    s = offset(start, head)
    e = offset(end, tail)

    audio = audio[s:e]
    if fade_in:
        audio = audio.fade_in(tomsecs(fade_in))
    if fade_out:
        audio = audio.fade_out(tomsecs(fade_out))

    return audio, len(audio) == ln
