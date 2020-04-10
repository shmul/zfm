import pydub
import pydub.playback
import os
import pathlib


def playall(head: float, tail: float, fade_in: float, fade_out: float,
            files: str):
    for f in files:
        f = os.path.realpath(f)
        audio = pydub.AudioSegment.from_file(f)
        ln = len(audio)
        print((head * 1000), (ln - tail * 1000))
        audio = audio[(head * 1000):(ln - tail * 1000)]
        if fade_in:
            audio = audio.fade_in(int(fade_in * 1000))
        if fade_out:
            audio = audio.fade_out(int(fade_out * 1000))

        pydub.playback.play(audio)
