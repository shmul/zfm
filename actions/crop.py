import pydub
import pydub.playback
import pydub.utils
import typing
import csv
import os
from actions.prepare import prepare
from actions.m3uparser import parsem3u

from datetime import timedelta
import urllib.parse

AudioFiles = typing.NewType('AudioFiles', typing.List[str])

csv_field_names = [
    'file', 'start', 'end', 'head', 'tail', 'fade_in', 'fade_out'
]


def at_targe_dir(file: str, target_dir: str) -> str:
    dir = target_dir
    if not dir:
        dir = os.path.dirname(file)

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

    cropped = at_targe_dir(file, target_dir)
    target = os.path.join(cropped, os.path.basename(file))
    if not dry_run:
        if os.path.exists(target):
            os.remove(target)
        if identical:
            os.symlink(file, target)
        else:
            segment.export(target)

    return segment


def round_to_sec(td: timedelta) -> timedelta:
    return timedelta(seconds=round(td.seconds, 0))


def to_csv(m3ufile: str, target_dir: str):
    # if not m3ufile.endswith("m3u"):
    #     print("incompatible file")
    #     return
    basename = os.path.splitext(os.path.basename(m3ufile))[0]
    csvfile = os.path.join(at_targe_dir(m3ufile, target_dir),
                           basename + '.csv')

    playlist = parsem3u(m3ufile)

    f = open(csvfile, 'w')
    print("creating ", csvfile)

    with f:
        writer = csv.DictWriter(f, fieldnames=csv_field_names)
        writer.writeheader()
        for track in playlist:
            writer.writerow({'file': track.path})


def preview_track(preview: int,idx: int,track):
    audio = track['audio']
    if track.get('skip'):
        return

    print('\n==== {idx} [{len}] {artist} - {title}'.format(**track))
    pydub.playback.play(audio[:preview])
    pydub.playback.play(audio[-preview:])

#file,start,end,head,tail,fade_in,fade_out,
def crop_many(csvfile: str,
              target_dir: str,
              preview: float,
              just: int,
              one_by_one: bool,
              dry_run: bool = False) -> AudioFiles:
    tracks = []
    cropped = at_targe_dir(csvfile, target_dir)
    overalltime = 0
    preview = preview * 1000

    with open(csvfile, newline='') as file:
        for idx, row in enumerate(csv.DictReader(file)):
            if just >= 0 and idx != just:
                continue

            # another input option is for a list of `key=value` pairs
            kvmode = False
            record = row.copy()
            for k in row:
                if row[k] == None:
                    continue
                parts = row[k].split("=")
                if len(parts) == 2:
                    if not kvmode:  # we switch to kv mode so we need to clear all the existing value
                        kvmode = True
                        for kk in record:
                            record[kk] = None

                    key = parts[0].strip()
                    value = parts[1].strip()
                    if key == "" or value == "":
                        raise Exception(row[k])
                    else:
                        record[key] = value

            record['target_dir'] = cropped
            record['file'] = urllib.parse.unquote(
                urllib.parse.urlparse(row['file']).path)
            audio = crop(**record, dry_run=True)
            mi = pydub.utils.mediainfo(row['file'])
            tags = mi.get('TAG')
            if tags != None:
                artist = tags.get('ARTIST') or tags.get('artist')
                title = tags.get('TITLE') or tags.get('title')
                skip = False
                if artist == None:
                    artist = os.path.basename(row['file'])
            else:
                artist = os.path.basename(row['file'])
                title = None
                skip = True

            s = 0
            if audio:
                s = len(audio) / 1000

            ln = round_to_sec(timedelta(seconds=s))

            track = {
                "idx": str(idx).zfill(2),
                "audio": audio,
                "artist": artist,
                "title": title,
                "skip": skip,
                "len": str(round_to_sec(ln)),
            }
            tracks.append(track)
            line = '({acc}) [{len}] {artist} - {title}'
            if title == None or title == "":
                line = '({acc}) [{len}] {artist}'

            print(
                line.format(**track,
                            acc=round_to_sec(timedelta(seconds=overalltime))))
            overalltime += s

    playlist = pydub.AudioSegment.empty()

    with open(os.path.join(cropped, "tracklist.txt"), 'w') as tracklist:
        for track in tracks:
            if preview==0:
                playlist += track['audio']

            if track.get('skip'):
                continue
            tracklist.write('{artist} - {title}\n'.format(**track))

    print("overall time", str(round_to_sec(timedelta(seconds=overalltime))))

    if not dry_run and not one_by_one and just == -1:
        target = os.path.join(cropped, "playlist.mp3")
        print(target)
        playlist.export(
            target, format="mp3",
            bitrate="320k")  # can't use `parameters=["-c", "copy"]`

    if preview != 0:
        for idx, track in enumerate(tracks):
            preview_track(preview,idx,track)
