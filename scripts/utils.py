"""The helper functions.
"""

import numpy as np
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
        rc: The row count.
        cc: The column count.
    """
    rc = int(np.sqrt(fc))
    while fc % rc != 0:
        rc -= 1
    cc = int(fc/rc)
    return (rc, cc)
