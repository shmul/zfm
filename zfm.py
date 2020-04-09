import actions
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
              help='head position (in secs)',
              show_default=True)
@click.option('--tail',
              '-t',
              default=0,
              help='tail position (in secs from end of the track)',
              show_default=True)
@click.argument('filename', type=click.Path(exists=True))
def crop(start, end, head, tail, filename):
    '''Crop file'''
    click.echo('zfm crop')
    actions.crop(start, end, head, tail, filename)


@zfm.command()
def export():
    '''Export files'''
    click.echo('zfm cmd2')


zfm.add_command(crop)

if __name__ == '__main__':
    zfm()
