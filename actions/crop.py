import pydub
import pydub.playback
import typing
import csv
import os
from actions.prepare import prepare

AudioFiles = typing.NewType('AudioFiles', typing.List[str])


# start/end are of the format [hh:]mm:ss; head/tail are secs
def crop(file: str, start: str, end: str, head: float, tail: float,
         fade_in: float, fade_out: float, play: bool):
    segment = prepare(file,
                      start=start,
                      end=end,
                      head=head,
                      tail=tail,
                      fade_in=fade_in,
                      fade_out=fade_out)
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
