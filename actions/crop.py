import pydub
import typing
import datetime
import pathlib
import csv

Msecs = typing.NewType('Msecs', int)
AudioFiles = typing.NewType('AudioFiles', List[string])


def hhmmss(tstr: str) -> int:
    if not tstr:
        return 0

    t = datetime.strptime(tstr, '%H:%M:%S')
    if not t:
        t = datetime.strptime(tstr, '%M:%S')
    if not t:
        return 0

    return t.timestamp()


def offset(hms: string, sec: int) -> Msecs:
    s = hhmmss(hms)
    if s != 0:
        return Msecs(s * 1000)
    if ln == 0:
        return Msecs(sec * 1000)
    return Msecs(sec * 1000)


# start/end are of the format [hh:]mm:ss; head/tail are secs
def crop(file: str, start: str, end: string, head: int, tail: int):
    cropped = pydub.AudioSegment.from_file(pathlib.Path(file))
    ln = len(cropped)
    if tail == 0:
        tail = ln
    s = offset(start, head)
    e = offset(end, tail)

    return cropped[s:e]


def crop_many(csvfile: str) -> AudioFiles:
    audios = []
    with open(csvfile, newline='') as file:
        for row in csv.reader(file):
            audio = crop(row['file'], row['start'], row['end'], row['head'],
                         row['tail'])
            audios.append(audio)

    return crop(file, start, end, head, tail)
