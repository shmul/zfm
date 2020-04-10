import pydub
import pydub.playback
import os
import pathlib


def playall(head: float, tail: float, files: str):
    for f in files:
        f = os.path.realpath(f)
        audio = pydub.AudioSegment.from_file(f)
        ln = len(audio)
        if head == 0 and tail == 0:
            pydub.playback.play(audio)
            continue

        if head > 0:
            pydub.playback.play(audio[:head * 1000])

        if tail > 0:
            pydub.playback.play(audio[ln - tail * 1000:])
