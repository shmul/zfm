import actions.play
import actions.crop
import click


@click.group()
def zfm():
    pass


@zfm.command()
@click.option('--start',
              '-s',
              type=str,
              help='start position (format [hh:]mm:ss)',
              show_default=True)
@click.option('--end',
              '-e',
              type=str,
              help='start position (format [hh:]mm:ss)',
              show_default=True)
@click.option('--head',
              '-h',
              default=0,
              type=float,
              help='head position (in secs)',
              show_default=True)
@click.option('--tail',
              '-t',
              default=0,
              type=float,
              help='tail position (in secs from end of the track)',
              show_default=True)
@click.option('--fade-in',
              '-i',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.option('--fade-out',
              '-o',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.option('--play/--no-play',
              '-p',
              default=False,
              help='play instead of save',
              show_default=True)
@click.argument('filename', type=click.Path(exists=True))
def crop(start, end, head, tail, fade_in, fade_out, play, filename):
    '''Crop file'''
    click.echo('zfm crop')
    actions.crop.crop(filename, start, end, head, tail, fade_in, fade_out,
                      play)


@zfm.command()
@click.option('--head',
              '-h',
              default=0,
              type=float,
              help='head position (in secs)',
              show_default=True)
@click.option('--tail',
              '-t',
              default=0,
              type=float,
              help='tail position (in secs from end of the track)',
              show_default=True)
@click.option('--fade-in',
              '-i',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.option('--fade-out',
              '-o',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.argument('files', nargs=-1)
def play(head, tail, fade_in, fade_out, files):
    '''Play files'''
    actions.play.playall(head, tail, fade_in, fade_out, files)


zfm.add_command(crop)

if __name__ == '__main__':
    zfm()
