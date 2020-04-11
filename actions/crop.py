import pydub
import pydub.playback
import pydub.utils
import typing
import csv
import os
from actions.prepare import prepare
from datetime import timedelta

AudioFiles = typing.NewType('AudioFiles', typing.List[str])


def cropped_dir(file: str, target_dir: str) -> str:
    dir = target_dir
    if not dir:
        dir = os.path.join(os.path.dirname(file), "cropped")

    os.makedirs(dir, exist_ok=True)
    return dir


# start/end are of the format [hh:]mm:ss; head/tail are secs
def crop(file: str,
         start: str,
         end: str,
         head: float,
         tail: float,
         fade_in: float,
         fade_out: float,
         play: bool = False,
         target_dir: str = None,
         dry_run: bool = False):
    segment, identical = prepare(file,
                                 start=start,
                                 end=end,
                                 head=head,
                                 tail=tail,
                                 fade_in=fade_in,
                                 fade_out=fade_out)
    if play:
        pydub.playback.play(segment)
        return

    cropped = cropped_dir(file, target_dir)
    target = os.path.join(cropped, os.path.basename(file))
    if os.path.exists(target):
        os.remove(target)
    if not dry_run:
        if identical:
            os.symlink(file, target)
        else:
            segment.export(target)

    return segment


def round_to_sec(td: timedelta) -> timedelta:
    return timedelta(seconds=round(td.seconds, 0))


#file,start,end,head,tail,fade_in,fade_out,
def crop_many(csvfile: str,
              target_dir: str,
              dry_run: bool = False) -> AudioFiles:
    tracks = []
    cropped = cropped_dir(csvfile, target_dir)
    overalltime = 0
    with open(csvfile, newline='') as file:
        for row in csv.DictReader(file):
            row['target_dir'] = cropped
            audio = crop(**row, dry_run=True)
            tags = pydub.utils.mediainfo(row['file'])['TAG']
            artist = tags.get('ARTIST') or tags.get('artist')
            title = tags.get('TITLE') or tags.get('title')
            s = 0
            if audio:
                s = len(audio) / 1000

            ln = round_to_sec(timedelta(seconds=s))

            track = {
                "audio": audio,
                "artist": artist,
                "title": title,
                "len": str(round_to_sec(ln)),
            }
            tracks.append(track)
            print('{artist} - {title} [{len}] (start: {acc})'.format(
                **track, acc=round_to_sec(timedelta(seconds=overalltime))))
            overalltime += s

    playlist = pydub.AudioSegment.empty()
    with open(os.path.join(cropped, "tracklist.txt"), 'w') as tracklist:
        for track in tracks:
            playlist += track['audio']
            if not dry_run:
                tracklist.write('{artist} - {title}\n'.format(**track))

    target = os.path.join(cropped, "playlist.mp3")
    print(target)
    print("overall time", str(round_to_sec(timedelta(seconds=overalltime))))
    if not dry_run:
        playlist.export(target)
