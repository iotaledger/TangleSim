"""The simulation script to run multiverse-simulation in batch.
"""
import sys
import shutil
import textwrap
import subprocess
import time
import json

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
              -rp 'YOUR_RESULT_PATH_1' -fop 'YOUR_FIGURE_OUTPUT_PATH_1' -exec 'go run . --parentsCount=2' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_2' -fop 'YOUR_FIGURE_OUTPUT_PATH_2' -exec 'go run . --parentsCount=4; -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_3' -fop 'YOUR_FIGURE_OUTPUT_PATH_3' -exec 'go run . --parentsCount=8' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_4' -fop 'YOUR_FIGURE_OUTPUT_PATH_4' -exec 'go run . --parentsCount=16' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_5' -fop 'YOUR_FIGURE_OUTPUT_PATH_5' -exec 'go run . --parentsCount=32' -rt 100 -st DS
            - python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1
              -rp 'YOUR_RESULTS_PATH_6' -fop 'YOUR_FIGURE_OUTPUT_PATH_6' -exec 'go run . --parentsCount=64' -rt 100 -st DS
    """), formatter_class=argparse.RawTextHelpFormatter)

    parser.add_argument("-msp", "--MULTIVERSE_PATH", type=str,
                        help="The path of multiverse-simulation",
                        default=config.cd['MULTIVERSE_PATH'])
    parser.add_argument("-rp", "--RESULTS_PATH", type=str,
                        help="The path to save the simulation results",
                        default=config.cd['RESULTS_PATH'])
    parser.add_argument("-fop", "--GENERAL_FIGURE_OUTPUT_PATH", type=str,
                        help="The path to output the figures",
                        default=config.cd['GENERAL_FIGURE_OUTPUT_PATH'])
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
                        help="The slowdown factors for each variation. If only one element, then it will be used for all runs",
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
    parser.add_argument("-nc", "--NODES_COUNT", dest='NODES_COUNT',
                        help="Nodes count",
                        default=config.cd['NODES_COUNT'])

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

    # Check the slowdown factors settings
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
    # os.makedirs(config.cd['FIGURE_OUTPUT_PATH'], exist_ok=True)

    # Run the simulation
    print(config.cd)
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
                # TPS = [100, 4000]
                for i, v in enumerate(vv):
                    if vn == 'zipfParameter' or vn == 'payloadLoss':
                        v = float(v)
                    else:
                        v = int(v)
                    os.system(
                        # f'{exec} --simulationTarget={target} --{vn}={v} --slowdownFactor={df[i]}')
                        f'{exec} --{vn}={v} --slowdownFactor={df[i]}')
            elif var == 'D':
                for i, v in enumerate(vv):
                    os.system(
                        f'{exec} --simulationTarget={target} --minDelay={float(v)} --maxDelay={float(v)} -slowdownFactor={df[i]}')
            elif var == 'AW':
                for i, v in enumerate([(0.66, True), (0.75, True), (0.5, False), (0.5, True)]):
                    os.system(
                        f'{exec} --simulationTarget={target} --confirmationThreshold={v[0]} --confirmationThresholdAbsolute={v[1]} -slowdownFactor={df[i]}')
            elif var == 'IM':
                for i, v in enumerate(vv):
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Accidental -accidentalMana="{v}" -slowdownFactor={df[i]}')
            elif var == 'AD':
                for i, v in enumerate(vv):
                    v = str(float(v)/2)
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversaryMana="{v} {v}" -adversaryType="{adv_strategy}" -adversaryInitColors="R B" -slowdownFactor={df[i]}')
            elif var == 'AC':
                for i, v in enumerate(vv):
                    if "adversaryMana" not in exec:
                        logging.error(
                            f'You must specify "-adversaryMana" parameter!')
                        sys.exit(2)

                    v = str(int(float(v)/2))
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversaryNodeCounts="{v} {v}" -adversaryType="1 1" -adversaryInitColors="R B" -slowdownFactor={df[i]}')
            elif var == 'BS':
                for i, v in enumerate(vv):
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversaryMana="{v}" -slowdownFactor={df[i]}')
            elif var == 'SU':
                for i, v in enumerate(vv):
                    os.system(
                        f'{exec} --simulationTarget={target}  -simulationMode=Adversary -adversarySpeedup="{v} {v}" -slowdownFactor={df[i]}')
            elif var == 'MB':
                post_fix = f'_{str(vv[0])}'
                # TODO: Fix the hacking codes
                config.cd['SCRIPT_START_TIME'] += post_fix
                config.cd['GENERAL_FIGURE_OUTPUT_PATH'] = (
                    f"{config.cd['MULTIVERSE_PATH']}/results/{config.cd['SCRIPT_START_TIME']}/general/figures")

                cmd = f"{exec} -scriptStartTime={config.cd['SCRIPT_START_TIME']}"
                expire = 1020
                try:
                    result = subprocess.run(
                        cmd,
                        shell=True,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE,
                        timeout=expire  # 17 minutes
                    )
                    print(result.stdout)
                except subprocess.TimeoutExpired:
                    print(
                        f"Process timed out after {expire} seconds. Terminating simulation process.")

                # os.system(
                #     f"{exec} -scriptStartTime={config.cd['SCRIPT_START_TIME']}")
            else:
                logging.error(f'The VARIATIONS {var} is not supported!')
                sys.exit(2)

    # move_results(sim_result_path, folder)

    # Plot the figures
    if config.cd['PLOT_FIGURES']:

        # New ADDED
        plotter = FigurePlotter(config.cd)
        (n, t_confirmation, t_convergence, t_flips,
         t_unconfirming, t_depth) = c.FIGURE_NAMING_DICT[var]
        iter_suffix = ''
        folder = config.cd['GENERAL_OUTPUT_PATH']
        # plotter.confirmation_time_violinplot(n, folder + '/aw0*csv', f'CT_{n}_aw0_ct{iter_suffix}.pdf', t_confirmation, n)
        # plotter.confirmation_time_violinplot(n, folder + '/aw99*csv', f'CT_{n}_aw99_ct{iter_suffix}.pdf', t_confirmation, n)
        if config.cd['PLOT_VARIED_FIGURES']:
            plotter.plot_varied_block_information_violinplot(
                '', config.cd['VARIED_PATHS'], f'blockInformation.pdf', t_confirmation, config.cd['VARIED_LABELS'])
        else:
            plotter.plot_varied_block_information_violinplot(
                n, [folder + '/BlockInformation.csv'], f'blockInformation.pdf', t_confirmation, n)
            plotter.acceptance_delay_violinplot(
                n, folder + '/acceptanceTimeLatencyAmongNodes.csv', f'acceptanceTimeLatencyAmongNodes.pdf', t_confirmation, n)

        # TODO: Fix bug
        with open(config.cd['CONFIGURATION_PATH']) as f:
            c = json.load(f)

        config.update('SchedulerType', c['SchedulerType'])
        config.update('BURN_POLICIES', c['BurnPolicies'])

        weights = parse_int_node_attributes(
            f'{config.cd["SCHEDULER_OUTPUT_PATH"]}/weights.csv', config.cd)
        config.update('WEIGHTS', weights)

        localMetricNames = parse_metric_names(
            f'{config.cd["GENERAL_OUTPUT_PATH"]}/localMetrics.csv')
        for name in localMetricNames:
            if name != 'RMC':
                continue
            data, times = parse_per_node_metrics(
                f'{config.cd["GENERAL_OUTPUT_PATH"]}/{name}.csv')
            plot_per_node_metric(data, times, config.cd, name, "")
            plot_per_node_wo_spammer_metric(data, times, config.cd, name, "")

        # plot dissemination rates
        messages, times = parse_per_node_metrics(
            f'{config.cd["SCHEDULER_OUTPUT_PATH"]}/disseminatedMessages.csv')

        plot_total_traffic(messages, times, config.cd, "Traffic")

        # TODO: refine plotting scripts
        # plot number of partially confirmed blocks
        messages, times = parse_per_node_metrics(
            f'{config.cd["SCHEDULER_OUTPUT_PATH"]}/partiallyConfirmedMessages.csv')
        plot_per_node_metric(messages, times, config.cd,
                             "Partially Confirmed Blocks", "Number of Blocks")

        readyLengths, times = parse_per_node_metrics(
            f'{config.cd["GENERAL_OUTPUT_PATH"]}/Ready Lengths.csv')
        plot_per_node_metric(readyLengths, times, config.cd,
                             "Ready Lengths", "Number of Blocks")
        nonReadyLengths, times = parse_per_node_metrics(
            f'{config.cd["GENERAL_OUTPUT_PATH"]}/Non Ready Lengths.csv')
        plot_per_node_metric(nonReadyLengths, times, config.cd,
                             "Non Ready Lengths", "Number of Blocks")

        sys.exit(0)

        # update the configuration dictionary
        newconfigs = parse_config(
            config.cd['RESULTS_PATH']+"/"+config.cd['SCRIPT_START_TIME']+"/config.csv")

        # copy the config.go file
        source_path = "./config/config.go"
        destination_dir = os.path.join(
            config.cd['RESULTS_PATH'], config.cd['SCRIPT_START_TIME'])
        destination_path = os.path.join(destination_dir, "config.go")
        shutil.copyfile(source_path, destination_path)

        os.makedirs(config.cd['GENERAL_FIGURE_OUTPUT_PATH'], exist_ok=True)
        print(
            f"Generating figures to {config.cd['GENERAL_FIGURE_OUTPUT_PATH']}")

        os.makedirs(config.cd['SCHEDULER_FIGURE_OUTPUT_PATH'], exist_ok=True)
        print(
            f"Generating figures to {config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']}")

        for k in newconfigs:
            config.update(k, newconfigs[k])
        burnPolicies = parse_int_node_attributes(
            config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/burnPolicies.csv", config.cd)
        config.update('BURN_POLICIES', burnPolicies)
        weights = parse_int_node_attributes(
            config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/weights.csv", config.cd)
        config.update('WEIGHTS', weights)
        # plot dissemination rates
        messages, times = parse_per_node_metrics(
            config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/disseminatedMessages.csv")
        plot_per_node_rates(messages, times, config.cd, "Dissemination Rates")
        plot_total_rate(messages, times, config.cd, "Total Dissemination Rate")
        # plot confirmation rates
        messages, times = parse_per_node_metrics(
            config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/fullyConfirmedMessages.csv")
        plot_per_node_rates(messages, times, config.cd, "Confirmation Rates")
        plot_total_rate(messages, times, config.cd, "Total Confirmation Rate")
        plot_total_rate(messages, times, config.cd,
                        "Total Confirmation Rate", 200)
        # # plot number of partially confirmed blocks
        # messages, times = parse_per_node_metrics(
        #     config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/partiallyConfirmedMessages.csv")
        # plot_per_node_metric(messages, times, config.cd,
        #                      "Partially Confirmed Blocks", "Number of Blocks")
        # plot number of unconfirmed blocks
        messages, times = parse_per_node_metrics(
            config.cd['GENERAL_FIGURE_OUTPUT_PATH']+"/unconfirmedMessages.csv")
        plot_per_node_metric(messages, times, config.cd,
                             "Unconfirmed Blocks", "Number of Blocks")
        # plot number of undisseminated blocks
        messages, times = parse_per_node_metrics(
            config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/undisseminatedMessages.csv")
        plot_per_node_metric(messages, times, config.cd,
                             "Undisseminated Blocks", "Number of Blocks")
        # plot dissemination latencies
        latencies = parse_latencies(
            config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/DisseminationLatency.csv", config.cd)
        plot_latency_cdf(latencies, config.cd, "Dissemination Latency")
        # plot confirmation latencies
        latencies = parse_latencies(
            config.cd['SCHEDULER_FIGURE_OUTPUT_PATH']+"/ConfirmationLatency.csv", config.cd)
        plot_latency_cdf(latencies, config.cd, "Confirmation Latency")
        # plot local metrics
        localMetricNames = parse_metric_names(
            config.cd['GENERAL_FIGURE_OUTPUT_PATH']+"/localMetrics.csv")
        for name in localMetricNames:
            data, times = parse_per_node_metrics(
                config.cd['GENERAL_FIGURE_OUTPUT_PATH']+"/" + name + ".csv")
            plot_per_node_metric(data, times, config.cd, name, "")
            plot_per_node_wo_spammer_metric(data, times, config.cd, name, "")

        plot_traffic(pd.read_csv(
            config.cd['GENERAL_FIGURE_OUTPUT_PATH']+"/Traffic.csv"), "Traffic",  config.cd)

        readyLengths, times = parse_per_node_metrics(
            f'{config.cd["SCHEDULER_OUTPUT_PATH"]}/readyLengths.csv')
        plot_per_node_metric(readyLengths, times, config.cd,
                             "Ready Lengths", "Number of Blocks")
        nonReadyLengths, times = parse_per_node_metrics(
            f'{config.cd["SCHEDULER_OUTPUT_PATH"]}/nonReadyLengths.csv')
        plot_per_node_metric(nonReadyLengths, times, config.cd,
                             "Non Ready Lengths", "Number of Blocks")
