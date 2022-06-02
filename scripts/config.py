import numpy as np
from pathlib import Path

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
        self.cd['FIGURE_OUTPUT_PATH'] = (
            self.cd['MULTIVERSE_PATH'] + '/scripts/figures')

        # The output folder suffix (e.g., ct for confirmation time and ds for double spending)
        self.cd['SIMULATION_TARGET'] = 'CT'

        # The variations to run
        # N, K, S, D (Number of nodes/parents, Zipfs, delays)
        self.cd['VARIATIONS'] = 'N'

        # The variations value list
        self.cd['VARIATION_VALUES'] = list(range(100, 1001, 100))

        # The deceleration factor list
        self.cd['DECELERATION_FACTORS'] = [1, 2, 2, 3, 5, 10, 15, 20, 25, 30]

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
        self.cd['RUN_SIM'] = False
        self.cd['PLOT_FIGURES'] = False

        # Adversary strategies
        self.cd["ADVERSARY_STRATEGY"] = "1 1"

    def update(self, k, v):
        """Update the key/value pair of the configuration.

        Args:
            k: The configuration key.
            v: The configuration value.
        """
        self.cd[k] = v
