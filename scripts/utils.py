"""The helper functions.
"""

import logging
import os


def move_results(src, dst):
    """Move the files from the source folder to the destination folder.

    Args:
        src: The source folder.
        dst: The destination folder.
    """

    if not os.path.isdir(dst):
        os.mkdir(dst)
    logging.info(f'Moving folder {src} to {dst}...')
    os.system(f'mv {src}/*.config {dst}')
    os.system(f'mv {src}/*.csv {dst}')


def get_row_col_counts(fc):
    """Return the row/columns counts of the figure.

    Args:
        fc: The figure count.

    Returns:
        rn: The number of rows.
        cn: The number of columns.
    """
    rn = 4
    cn = 4
    if fc == 6:
        rn = 2
        cn = 3
    elif fc == 10:
        rn = 2
        cn = 5
    elif fc <= 12:
        rn = 3
        cn = 4
    return (rn, cn)
