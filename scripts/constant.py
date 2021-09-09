"""The constant/configuration for the simulation script.
"""

# Data paths
MULTIVERSE_PATH = '/home/piotrek/Documents/iota/multiverse-simulation'
RESULTS_PATH = MULTIVERSE_PATH + "/results"

FIGURE_OUTPUT_PATH = MULTIVERSE_PATH + '/scripts/figures'

# The output folder suffix (e.g., ct for confirmation time and ds for double spending)
SIMULATION_TARGET = 'DS'

# Execution way (e.g., 'go run .' or './multiverse_sim')
EXECUTE = './multiverse_sim'

# Transparent figure
TRANSPARENT = False

# The begining x_axis in ns
X_AXIS_BEGIN = 20000_000_000

# The timescale of the 'ns after start' is ns. Use sec as the unit time.
ONE_SEC = 1000_000_000

# The issuance time of colored message (in ns)
COLORED_MSG_ISSUANCE_TIME = 2000_000_000

# Flags of operations
RUN_SIM = True
PLOT_FIGURES = True

# Define the list of styles
CLR_LIST = ['k', 'b', 'g', 'r']  # list of basic colors
STY_LIST = ['-', '--', '-.', ':']  # list of basic linestyles

# Define the target to parse
TARGET = "Confirmation Time (ns)"
ISSUED_MESSAGE = "# of Issued Messages"

# Rename the parameters
VAR_DICT = {'TipsCount': 'k', 'ZipfParameter': 's', 'NodesCount': 'N'}

# Items for double spending figures
COLORED_CONFIRMED_LIKE_ITEMS = [
    'Blue (Confirmed)', 'Red (Confirmed)', 'Blue (Like)', 'Red (Like)']

# The color list for the double spending figures
DS_CLR_LIST = ['b', 'r', 'b', 'r']
DS_STY_LIST = ['-', '-', '--', '--']
