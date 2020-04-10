import pydub
import pydub.playback
import typing
import pytimeparse
import pathlib
import csv
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


# start/end are of the format [hh:]mm:ss; head/tail are secs
def crop(file: str, start: str, end: str, head: float, tail: float,
         play: bool):
    file = os.path.realpath(file)
    cropped = pydub.AudioSegment.from_file(pathlib.Path(file))
    ln = len(cropped)
    if tail == 0:
        tail = ln / 1000
    s = offset(start, head)
    e = offset(end, tail)
    print("file {}, len: {}, start: {}, end: {}".format(file, ln, s, e))
    segment = cropped[s:e]
    if play:
        pydub.playback.play(segment)
        return

    cropped = os.path.join(os.path.dirname(file), "cropped")
    os.makedirs(cropped, exist_ok=True)
    target = os.path.join(cropped, os.path.basename(file))
    if os.path.exists(target):
        os.remove(target)
    if s == 0 and e == ln:
        print("softlink")
        os.symlink(file, target)
    else:
        segment.export(target)


def crop_many(csvfile: str) -> AudioFiles:
    audios = []
    with open(csvfile, newline='') as file:
        for row in csv.reader(file):
            audio = crop(row['file'], row['start'], row['end'], row['head'],
                         row['tail'], False)
            audios.append(audio)

    return crop(file, start, end, head, tail)
