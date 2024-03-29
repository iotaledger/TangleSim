import numpy as np
from pathlib import Path
from datetime import datetime

"""The configuration for the simulation script.
"""


class Configuration:
    """The configuration of simulation
    """

    def __init__(self):
        """Initialize the default configuration values

        """
        # The configuration dictionary
        self.cd = {}

        # The data paths
        self.cd['MULTIVERSE_PATH'] = str(Path().absolute().parent)
        self.cd['RESULTS_PATH'] = self.cd['MULTIVERSE_PATH'] + "/results"
        self.cd['SCRIPT_START_TIME'] = datetime.strftime(
            datetime.now(), "%Y%m%d_%H%M")
        # self.cd['SCRIPT_START_TIME'] = '20230715_0555'
        self.cd['CONFIGURATION_PATH'] = f"{self.cd['MULTIVERSE_PATH']}/results/{self.cd['SCRIPT_START_TIME']}/mb.config"
        self.cd['GENERAL_OUTPUT_PATH'] = f"{self.cd['MULTIVERSE_PATH']}/results/{self.cd['SCRIPT_START_TIME']}/general"
        self.cd['SCHEDULER_OUTPUT_PATH'] = f"{self.cd['MULTIVERSE_PATH']}/results/{self.cd['SCRIPT_START_TIME']}/scheduler"
        self.cd['GENERAL_FIGURE_OUTPUT_PATH'] = f"{self.cd['MULTIVERSE_PATH']}/results/{self.cd['SCRIPT_START_TIME']}/general/figures"
        self.cd['SCHEDULER_FIGURE_OUTPUT_PATH'] = f"{self.cd['MULTIVERSE_PATH']}/results/{self.cd['SCRIPT_START_TIME']}/scheduler/figures"

        #  (self.cd['MULTIVERSE_PATH'] + '/scripts/figures')

        self.cd['NODES_COUNT'] = 100
        # monitoring interval in milliseconds
        self.cd['MONITOR_INTERVAL'] = 100

        # The output folder suffix (e.g., ct for confirmation time and ds for double spending)
        self.cd['SIMULATION_TARGET'] = 'CT'

        # The variations to run
        # N, K, S, D, 'MB' (Number of nodes/parents, Zipfs, delays, manaburn policies)
        self.cd['VARIATIONS'] = 'N'

        # The variations value list
        # list of policies, separated by spaces
        self.cd['VARIATION_VALUES'] = [
            "2 0 2 2 2 2 2 2 2 2 2 0 2 2 2 2 2 2 2 2"]

        # The deceleration factor list
        self.cd['DECELERATION_FACTORS'] = [1]

        # The repetition of each variation
        self.cd['REPETITION_TIME'] = 1

        # Execution way (e.g., 'go run .' or './multiverse_sim')
        # EXECUTE = './multiverse_sim'
        self.cd['EXECUTE'] = 'go run .'

        # Transparent figure
        self.cd['TRANSPARENT'] = False

        # The begining x_axis in ns
        self.cd['X_AXIS_BEGIN'] = 20000_000_000

        # The issuance time of colored message (in ns)
        self.cd['COLORED_MSG_ISSUANCE_TIME'] = 2000_000_000

        # Flags of operations
        self.cd['RUN_SIM'] = True
        self.cd['PLOT_FIGURES'] = True

        # Adversary strategies
        self.cd["ADVERSARY_STRATEGY"] = "1 1"

        # Plotting variation setting
        self.cd['PLOT_VARIED_FIGURES'] = False
        self.cd['VARIED_PATHS'] = []
        self.cd['VARIED_LABELS'] = [
            # '15.07low',
            # '32.47low',
            # # '15.56high',
            # # '29.68high',
            # # '32.47low_opt'
        ]

    def update(self, k, v):
        """Update the key/value pair of the configuration.

        Args:
            k: The configuration key.
            v: The configuration value.
        """
        self.cd[k] = v
