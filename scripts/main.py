"""The simulation script to run multiverse-simulation in batch.
"""
import argparse
import logging
import os

import constant as c
from utils import *
from plotting import FigurePlotter
from parsing import FileParser
from config import Configuration


def parse_arg():
    """Parse the arguments and return the updated configuration dictionary.

    Returns:
        config: The configuration.
    """

    parser = argparse.ArgumentParser()

    parser.add_argument("-msp", "--MULTIVERSE_PATH", type=str,
                        help="The path of multiverse-simulation")
    parser.add_argument("-rp", "--RESULTS_PATH", type=str,
                        help="The path to save the simulation results")
    parser.add_argument("-fop", "--FIGURE_OUTPUT_PATH", type=str,
                        help="The path to output the figures")
    parser.add_argument("-st", "--SIMULATION_TARGET", type=str,
                        help="The simulation target, CT (confirmation time) or DS (double spending)")
    parser.add_argument("-rt", "--REPETITION_TIME", type=int,
                        help="The number of runs for a single configuration")
    parser.add_argument("-v", "--VARIATIONS",
                        help="# N, K, S, D (Number of nodes/parents, Zipfs, delays)")
    parser.add_argument("-vv", "--VARIATION_VALUES", nargs="+", type=float,
                        help="The variation values, e.g., '100 200 300' for different N")
    parser.add_argument("-df", "--DECELERATION_FACTORS", nargs="+", type=int,
                        help="The deceleration factors for each variation. If only one element, then it will be used for all runs")
    parser.add_argument("-exec", "--EXECUTE", type=str,
                        help="Execution way, e.g., 'go run .' or './multiverse_sim'")
    parser.add_argument("-t", "--TRANSPARENT", type=bool,
                        help="The generated figures should be transparent")
    parser.add_argument("-xb", "--X_AXIS_BEGIN", type=str,
                        help="The begining x axis in ns")
    parser.add_argument("-ct", "--COLORED_MSG_ISSUANCE_TIME", type=str,
                        help="The issuance time of colored message (in ns)")
    parser.add_argument("-rs", "--RUN_SIM", dest='RUN_SIM', action='store_true',
                        help="Run the simulation")
    parser.add_argument("-pf", "--PLOT_FIGURES", dest='PLOT_FIGURES', action='store_true',
                        help="Plot the figures")

    # Update the che configuration dictionary
    config = Configuration()
    args = parser.parse_args()
    for arg in vars(args):
        v = getattr(args, arg)
        if not (v is None):
            config.update(arg, v)

    return config


if __name__ == '__main__':
    debug_level = "INFO"
    logging.basicConfig(
        level=debug_level,
        format="%(asctime)s,%(msecs)d %(levelname)s: %(message)s",
        datefmt="%Y%M%D %H:%M:%S",
    )

    # Update the che configuration dictionary
    config = parse_arg()

    # Check the deceleration factors settings
    df = config.cd['DECELERATION_FACTORS']
    vv = config.cd['VARIATION_VALUES']
    if len(df) != 1 and (len(df) != len(vv)):
        logging.error(
            'The DECELERATION_FACTORS should be only one value, or the same length with the VARIATION_VALUES!')
        os.exit(1)
    elif len(df) == 1:
        df = df * len(vv)

    var = config.cd['VARIATIONS']
    exec = config.cd['EXECUTE']
    target = config.cd['SIMULATION_TARGET']
    result_path = config.cd['RESULTS_PATH']
    folder = f'{result_path}/var_{var}_{target}'

    os.makedirs(config.cd['RESULTS_PATH'], exist_ok=True)
    os.makedirs(config.cd['FIGURE_OUTPUT_PATH'], exist_ok=True)

    # Run the simulation
    if config.cd['RUN_SIM']:
        os.chdir(config.cd['MULTIVERSE_PATH'])
        if var in c.SIMULATION_VAR_DICT:
            # Simulation var
            vn = c.SIMULATION_VAR_DICT[var]
            for i, v in enumerate(vv):
                os.system(
                    f'{exec} --simulationTarget={target} --{vn}={v} --decelerationFactor={df[i]}')
        elif var == 'D':
            for i, v in vv:
                os.system(
                    f'{exec} --simulationTarget={target} --MinDelay={v} --maxDelay={v} -decelerationFactor={df[i]}')
        else:
            logging.error(f'The VARIATIONS {var} is not supported!')
            os.exit(2)

        move_results(result_path, folder)

    # Plot the figures
    if config.cd['PLOT_FIGURES']:
        plotter = FigurePlotter(config.cd)

        # (The variation name in the configuration file, the confirmation time figure title)
        n, t = c.FIGURE_NAMING_DICT[var]

        plotter.confirmation_time_plot(
            n, folder + '/aw*csv', 'CT_{n}.png', t, c.VAR_DICT[n])

        plotter.throughput_plot(n, folder + '/tp*csv',
                                'CT_{n}_tp.png', len(vv))

        plotter.confirmed_like_color_plot(
            n, folder + '/cc*csv', 'DS_{n}_cc.png', len(vv))
