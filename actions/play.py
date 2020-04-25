import pydub
import pydub.playback
from actions.prepare import prepare


def playall(head: float, tail: float, fade_in: float, fade_out: float,
            files: str):
    for f in files:
        audio, ln = prepare(file=f,
                            head=head,
                            tail=tail,
                            fade_in=fade_in,
                            fade_out=fade_out)

        pydub.playback.play(audio)
