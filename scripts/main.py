"""The simulation script to run multiverse-simulation in batch.
"""
import sys
import textwrap

import constant as c
from config import Configuration
from plotting import FigurePlotter
from utils import *


def parse_arg():
    """Parse the arguments and return the updated configuration dictionary.
    Returns:
        config: The configuration.
    """

    config = Configuration()

    parser = ArgumentParserWithDefaults(
        description=textwrap.dedent("""        
    In the following we provide some examples.
    - Different Ns for confirmation time (CT) analysis
        - The output results will be put in [RESULTS_PATH]/var_N_CT
        - Usage: python3 main.py -rs -pf
    - Different Ks for double spending (DS) analysis, 100 times
        - The output results will be put in [RESULTS_PATH]/var_K_DS
        - Usage: python3 main.py -rs -pf -v K -vv 2 4 8 16 32 64 -df 1 -rt 100 -st DS
    - Different Ss for double spending (DS) analysis, 100 times
        - The output results will be put in [RESULTS_PATH]/var_S_DS
        - NOTE: Need to use -rp, -fop to specify different RESULTS_PATH and FIGURE_OUTPUT_PATH
          if customized EXECUTE is set, or the output folder for the multiverse results will be
          put in the same folder, and the output figures will be overwritten.
        - Usage example of generating different Ss (0~2.2) and different Ks (2, 4, 8, 16, 32, 64)
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULT_PATH_1' -fop 'YOUR_FIGURE_OUTPUT_PATH_1' -exec 'go run . --tipsCount=2' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_2' -fop 'YOUR_FIGURE_OUTPUT_PATH_2' -exec 'go run . --tipsCount=4; -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_3' -fop 'YOUR_FIGURE_OUTPUT_PATH_3' -exec 'go run . --tipsCount=8' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_4' -fop 'YOUR_FIGURE_OUTPUT_PATH_4' -exec 'go run . --tipsCount=16' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_5' -fop 'YOUR_FIGURE_OUTPUT_PATH_5' -exec 'go run . --tipsCount=32' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_6' -fop 'YOUR_FIGURE_OUTPUT_PATH_6' -exec 'go run . --tipsCount=64' -rt 100 -st DS
    """), formatter_class=argparse.RawTextHelpFormatter)

    parser.add_argument("-msp", "--MULTIVERSE_PATH", type=str,
                        help="The path of multiverse-simulation",
                        default=config.cd['MULTIVERSE_PATH'])
    parser.add_argument("-rp", "--RESULTS_PATH", type=str,
                        help="The path to save the simulation results",
                        default=config.cd['RESULTS_PATH'])
    parser.add_argument("-fop", "--FIGURE_OUTPUT_PATH", type=str,
                        help="The path to output the figures",
                        default=config.cd['FIGURE_OUTPUT_PATH'])
    parser.add_argument("-st", "--SIMULATION_TARGET", type=str,
                        help="The simulation target, CT (confirmation time) or DS (double spending)",
                        default=config.cd['SIMULATION_TARGET'])
    parser.add_argument("-rt", "--REPETITION_TIME", type=int,
                        help="The number of runs for a single configuration",
                        default=config.cd['REPETITION_TIME'])
    parser.add_argument("-v", "--VARIATIONS",
                        help="N, K, S, D (Number of nodes, parents, Zipfs, delays)",
                        default=config.cd['VARIATIONS'])
    parser.add_argument("-vv", "--VARIATION_VALUES", nargs="+", type=str,
                        help="The variation values, e.g., '100 200 300' for different N",
                        default=config.cd['VARIATION_VALUES'])
    parser.add_argument("-df", "--DECELERATION_FACTORS", nargs="+", type=int,
                        help="The deceleration factors for each variation. If only one element, then it will be used for all runs",
                        default=config.cd['DECELERATION_FACTORS'])
    parser.add_argument("-exec", "--EXECUTE", type=str,
                        help="Execution way, e.g., 'go run .' or './multiverse_sim'",
                        default=config.cd['EXECUTE'])
    parser.add_argument("-t", "--TRANSPARENT", type=bool,
                        help="The generated figures should be transparent",
                        default=config.cd['TRANSPARENT'])
    parser.add_argument("-xb", "--X_AXIS_BEGIN", type=str,
                        help="The begining x axis in ns",
                        default=config.cd['X_AXIS_BEGIN'])
    parser.add_argument("-ct", "--COLORED_MSG_ISSUANCE_TIME", type=str,
                        help="The issuance time of colored message (in ns)",
                        default=config.cd['COLORED_MSG_ISSUANCE_TIME'])
    parser.add_argument("-rs", "--RUN_SIM", dest='RUN_SIM', action='store_true',
                        help="Run the simulation",
                        default=config.cd['RUN_SIM'])
    parser.add_argument("-pf", "--PLOT_FIGURES", dest='PLOT_FIGURES', action='store_true',
                        help="Plot the figures",
                        default=config.cd['PLOT_FIGURES'])
    parser.add_argument("-as", "--ADVERSARY_STRATEGY", dest='ADVERSARY_STRATEGY',
                        help="Adversary types",
                        default=config.cd['ADVERSARY_STRATEGY'])

    # Update the che configuration dictionary
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
        sys.exit(1)
    elif len(df) == 1:
        df = df * len(vv)

    # Note that if relative paths are set, then the base path will be MULTIVERSE_PATH
    os.chdir(config.cd['MULTIVERSE_PATH'])

    # Get the vars/folder locations
    var = config.cd['VARIATIONS']
    exec = config.cd['EXECUTE']
    target = config.cd['SIMULATION_TARGET']
    sim_result_path = config.cd['MULTIVERSE_PATH'] + '/results'
    result_path = config.cd['RESULTS_PATH']
    base_folder = f'{result_path}/var_{var}_{target}'
    repetition = config.cd['REPETITION_TIME']
    adv_strategy = config.cd['ADVERSARY_STRATEGY']

    # Generate the folders if they don't exist
    os.makedirs(result_path, exist_ok=True)
    os.makedirs(config.cd['FIGURE_OUTPUT_PATH'], exist_ok=True)

    # Run the simulation
    if config.cd['RUN_SIM']:
        folder = base_folder
        os.makedirs(folder, exist_ok=True)
        for iter in range(repetition):
            if repetition != 1:
                folder = base_folder + f'/iter_{iter}'
                os.makedirs(folder, exist_ok=True)

            if var in c.SIMULATION_VAR_DICT:
                # Simulation var
                vn = c.SIMULATION_VAR_DICT[var]
                for i, v in enumerate(vv):
                    if vn == 'zipfParameter':
                        v = float(v)
                    else:
                        v = int(v)
                    os.system(
                        f'{exec} --simulationTarget={target} --{vn}={v} --decelerationFactor={df[i]}')
            elif var == 'D':
                for i, v in vv:
                    os.system(
                        f'{exec} --simulationTarget={target} --MinDelay={float(v)} --maxDelay={float(v)} -decelerationFactor={df[i]}')
            elif var == 'AW':
                for i, v in enumerate([(0.66, True), (0.75, True), (0.5, False), (0.5, True)]):
                    os.system(
                        f'{exec} --simulationTarget={target} --weightThreshold={v[0]} --weightThresholdAbsolute={v[1]} -decelerationFactor={df[i]}')
            elif var == 'IM':
                for i, v in enumerate(vv):
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Accidental -accidentalMana="{v}" -decelerationFactor={df[i]}')
            elif var == 'AD':
                for i, v in enumerate(vv):
                    v = str(float(v)/2)
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversaryMana="{v} {v}" -adversaryType="{adv_strategy}" -adversaryInitColors="R B" -decelerationFactor={df[i]}')
            elif var == 'AC':
                for i, v in enumerate(vv):
                    if "adversaryMana" not in exec:
                        logging.error(
                            f'You must specify "-adversaryMana" parameter!')
                        sys.exit(2)

                    v = str(int(float(v)/2))
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversaryNodeCounts="{v} {v}" -adversaryType="1 1" -adversaryInitColors="R B" -decelerationFactor={df[i]}')
            elif var == 'BS':
                for i, v in enumerate(vv):
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversaryMana="{v}" -decelerationFactor={df[i]}')
            elif var == 'SU':
                for i, v in enumerate(vv):
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversarySpeedup="{v} {v}" -decelerationFactor={df[i]}')
            else:
                logging.error(f'The VARIATIONS {var} is not supported!')
                sys.exit(2)

            move_results(sim_result_path, folder)

    # Plot the figures
    if config.cd['PLOT_FIGURES']:
        plotter = FigurePlotter(config.cd)

        # (The variation name in the configuration file, the confirmation time figure title,
        #  the convergence time figure title, the flips title, the unconfirming count title,
        #  the confirmation weight depth figure title)
        (n, t_confirmation, t_convergence, t_flips,
         t_unconfirming, t_depth) = c.FIGURE_NAMING_DICT[var]

        folder = base_folder
        iter_suffix = ''
        for iter in range(repetition):

            if repetition != 1:
                folder = base_folder + f'/iter_{iter}'
                iter_suffix = f'_iter_{iter}'

            if target == 'DS':
                if repetition != 1 and iter == 0:
                    # The distribution plots of multiple iterations are ran only one time.
                    plotter.convergence_time_distribution_plot(
                        n, base_folder, f'DS_{n}_cv.png', len(vv), repetition, title=t_convergence)

                    plotter.flips_distribution_plot(
                        n, base_folder, f'DS_{n}_fl.png', len(vv), repetition, title=t_flips)

                    plotter.unconfirmed_count_distribution_plot(
                        n, base_folder, f'DS_{n}_uc.png', len(vv), repetition, title=t_unconfirming)

                    plotter.confirmation_depth_distribution_plot(
                        n, base_folder, f'DP_{n}_cd.png', len(vv), repetition, title=t_depth)

                plotter.confirmed_like_color_plot(
                    n, folder + '/cc*csv', f'DS_{n}_cc{iter_suffix}.png', len(vv))

            plotter.confirmation_time_plot(
                n, folder + '/aw*csv', f'CT_{n}_ct{iter_suffix}.png', t_confirmation, c.VAR_DICT[n])

            plotter.throughput_plot(n, folder + '/tp*csv',
                                    f'CT_{n}_tp{iter_suffix}.png', len(vv))
