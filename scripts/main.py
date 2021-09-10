"""The simulation script to run multiverse-simulation in batch.
"""
import argparse
import logging
import os
import sys

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
                        help="N, K, S, D (Number of nodes/parents, Zipfs, delays)")
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
    """ Running Examples
    Different N:
    $python3 main.py -rs -pf -msp 'YOUR_PATH'

    Different K, 100 times:
    $python3 main.py -rs -pf -v K -vv 2 4 8 16 32 64 -df 1 -msp 'YOUR_PATH' -rt 100 -st DS

    Different S:
    NOTE: ned to specify -rp, -exec, -fop

    $python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 
    -msp 'YOUR_PATH' -rp 'RESULT_PATH' -fop 'FIGURE_PATH' -exec 'go run . --tipsCount=2 -rt 100 -st DS

    $python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 
    -msp 'YOUR_PATH' -rp 'SPECIFC_PATH' -fop 'FIGURE_PATH' -exec 'go run . --tipsCount=4 -rt 100 -st DS

    $python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 
    -msp 'YOUR_PATH' -rp 'SPECIFC_PATH' -fop 'FIGURE_PATH' -exec 'go run . --tipsCount=8 -rt 100 -st DS

    $python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 
    -msp 'YOUR_PATH' -rp 'SPECIFC_PATH' -fop 'FIGURE_PATH' -exec 'go run . --tipsCount=16 -rt 100 -st DS

    $python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 
    -msp 'YOUR_PATH' -rp 'SPECIFC_PATH' -fop 'FIGURE_PATH' -exec 'go run . --tipsCount=32 -rt 100 -st DS

    $python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 
    -msp 'YOUR_PATH' -rp 'SPECIFC_PATH' -fop 'FIGURE_PATH' -exec 'go run . --tipsCount=64 -rt 100 -st DS
    """

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
        sys.exit(1)
    elif len(df) == 1:
        df = df * len(vv)

    var = config.cd['VARIATIONS']
    exec = config.cd['EXECUTE']
    target = config.cd['SIMULATION_TARGET']
    result_path = config.cd['RESULTS_PATH']
    base_folder = f'{result_path}/var_{var}_{target}'
    repetition = config.cd['REPETITION_TIME']

    os.makedirs(result_path, exist_ok=True)
    os.makedirs(config.cd['FIGURE_OUTPUT_PATH'], exist_ok=True)

    # Run the simulation
    if config.cd['RUN_SIM']:
        os.chdir(config.cd['MULTIVERSE_PATH'])
        for iter in range(repetition):
            if repetition != 1:
                folder = base_folder + f'/iter_{iter}'
                os.makedirs(folder, exist_ok=True)

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
                sys.exit(2)

            move_results(result_path, folder)

    # Plot the figures
    if config.cd['PLOT_FIGURES']:
        plotter = FigurePlotter(config.cd)

        # (The variation name in the configuration file, the confirmation time figure title,
        #  the convergence time figure title)
        n, t_confirmation, t_convergence = c.FIGURE_NAMING_DICT[var]

        for iter in range(repetition):
            if repetition != 1:
                folder = base_folder + f'/iter_{iter}'
                plotter.convergence_time_distribution_plot(
                    n, base_folder, f'DS_{n}_cv.png', 2, repetition, title=t_convergence)

            plotter.confirmation_time_plot(
                n, folder + '/aw*csv', f'CT_{n}_{iter}.png', t_confirmation, c.VAR_DICT[n])

            plotter.throughput_plot(n, folder + '/tp*csv',
                                    f'CT_{n}_tp_{iter}.png', len(vv))

            plotter.confirmed_like_color_plot(
                n, folder + '/cc*csv', f'DS_{n}_cc_iter_{iter}.png', len(vv))
