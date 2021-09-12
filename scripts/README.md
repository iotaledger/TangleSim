# Multiverse simulation scripts

## About

Batch running the multiverse-simulation in different parameter sets, and generate the figures automatically.

## Requirements

- Install Python3.6+ from the [official website](https://www.python.org/downloads/)
- Install the required packages
```s
pip install -r requirements.txt
```

## Supported Arguments
- Note that the default MULTIVERSE_PATH will be the parent's folder of the `scripts` folder.
- Please use `python3 main.py -h` to see the default paths in your environments.
```s
optional arguments:
  -h, --help            show this help message and exit
  -msp MULTIVERSE_PATH, --MULTIVERSE_PATH MULTIVERSE_PATH
                        The path of multiverse-simulation
                        Default: [MULTIVERSE_PATH]
  -rp RESULTS_PATH, --RESULTS_PATH RESULTS_PATH
                        The path to save the simulation results
                        Default: [MULTIVERSE_PATH]/results
  -fop FIGURE_OUTPUT_PATH, --FIGURE_OUTPUT_PATH FIGURE_OUTPUT_PATH
                        The path to output the figures
                        Default: [MULTIVERSE_PATH]/scripts/figures
  -st SIMULATION_TARGET, --SIMULATION_TARGET SIMULATION_TARGET
                        The simulation target, CT (confirmation time) or DS (double spending)
                        Default: CT
  -rt REPETITION_TIME, --REPETITION_TIME REPETITION_TIME
                        The number of runs for a single configuration
                        Default: 1
  -v VARIATIONS, --VARIATIONS VARIATIONS
                        N, K, S, D (Number of nodes, parents, Zipfs, delays)
                        Default: N
  -vv VARIATION_VALUES [VARIATION_VALUES ...], --VARIATION_VALUES VARIATION_VALUES [VARIATION_VALUES ...]
                        The variation values, e.g., '100 200 300' for different N
                        Default: [100, 200, 300, 400, 500, 600, 700, 800, 900, 1000]
  -df DECELERATION_FACTORS [DECELERATION_FACTORS ...], --DECELERATION_FACTORS DECELERATION_FACTORS [DECELERATION_FACTORS ...]
                        The deceleration factors for each variation. If only one element, then it will be used for all runs
                        Default: [1, 2, 2, 3, 5, 10, 15, 20, 25, 30]
  -exec EXECUTE, --EXECUTE EXECUTE
                        Execution way, e.g., 'go run .' or './multiverse_sim'
                        Default: go run .
  -t TRANSPARENT, --TRANSPARENT TRANSPARENT
                        The generated figures should be transparent
                        Default: False
  -xb X_AXIS_BEGIN, --X_AXIS_BEGIN X_AXIS_BEGIN
                        The begining x axis in ns
                        Default: 20000000000
  -ct COLORED_MSG_ISSUANCE_TIME, --COLORED_MSG_ISSUANCE_TIME COLORED_MSG_ISSUANCE_TIME
                        The issuance time of colored message (in ns)
                        Default: 2000000000
  -rs, --RUN_SIM        Run the simulation
                        Default: False
  -pf, --PLOT_FIGURES   Plot the figures
                        Default: False
```

## Running examples
- Different Ns for confirmation time (CT) analysis
    - The output results will be put in `RESULTS_PATH`/var_N_CT
    - Usage
```s
python3 main.py -rs -pf
```

- Different Ks for double spending (DS) analysis, 100 times
    - The output results will be put in `RESULTS_PATH`/var_K_DS
    - Usage
```s
python3 main.py -rs -pf -v K -vv 2 4 8 16 32 64 -df 1 -rt 100 -st DS
```

- Different Ss for double spending (DS) analysis, 100 times
    - The output results will be put in `RESULTS_PATH`/var_S_DS
    - NOTE: Need to use -rp, -fop to specify different RESULTS_PATH and FIGURE_OUTPUT_PATH
      if customized EXECUTE is set, or the output folder for the multiverse results will be
      put in the same folder, and the output figures will be overwritten.
    - Usage example of generating different Ss (0~2.2) and different Ks (2, 4, 8, 16, 32, 64)
```s
python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 -rp 'k_2' -fop 'k_2/figures' -exec 'go run . --tipsCount=2' -rt 100 -st DS

python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 -rp 'k_4' -fop 'k_4/figures' -exec 'go run . --tipsCount=4' -rt 100 -st DS

python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 -rp 'k_8' -fop 'k_8/figures' -exec 'go run . --tipsCount=8' -rt 100 -st DS

python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 -rp 'k_16' -fop 'k_16/figures' -exec 'go run . --tipsCount=16' -rt 100 -st DS

python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 -rp 'k_32' -fop 'k_32/figures' -exec 'go run . --tipsCount=32' -rt 100 -st DS

python3 main.py -rs -pf -v S -vv 0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0 2.2 -df 1 -rp 'k_64' -fop 'k_64/figures' -exec 'go run . --tipsCount=64' -rt 100 -st DS
```