/* -------------------------------------------------------------------------- */
/*                             Glue for ES modules                            */
/* -------------------------------------------------------------------------- */
import path from 'path';
import { fileURLToPath } from 'url';
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/* -------------------------------------------------------------------------- */
/*                              standard imports                              */
/* -------------------------------------------------------------------------- */
import { execa, ExecaChildProcess, ExecaReturnValue, Options } from 'execa';
import { existsSync as fsExists } from 'fs';

/**
 * CommandOptions defines an interface for configuring subprocess signal and kill timeout values.
 */
interface CommandOptions {
    readonly killTimeout: number;
    readonly signalTimeout: number;
}

/**
 * Command defines an interface for congfiguring a subprocess command to pass to execute via the `run` function.
 */
interface Command {
    readonly executable: path.ParsedPath;
    readonly arguments?: string[];
    readonly options: CommandOptions;
}

/**
 * ExecaErrorIface adds additional properties to Error. We need to cast Error to a class that implements this interface.
 */
interface ExecaErrorIface {
    stdout: string;
}

/**
 * ExecaError provides a concrete implementation of an Error subclass that implements ExecaErrorIface.
 */
class ExecaError extends Error implements ExecaErrorIface {
    public stdout: string;

    public constructor() {
        super();
        this.stdout = '';
    }
}

/**
 * signalTimeout represents the time (in ms) to wait before sending the specified signal to the child process.
 */
const signalTimeout: number = 3000;

/**
 * processKillTimeout represents the time (in ms) to wait before forecfully terminating the child process. In the event
 * that the sent signal is not received or acted on by the child process, this timeout will ensure that the child will
 * continue to exit in a reasonable amount of time.
 */
const processKillTimeout: number = 5000;

/**
 * execaConfig encapsulate configuration options for the execa command.
 */
const execaConfig: Options = {};

/**
 * scriptPath represents the full path to the script that will be run as the child process.
 */
const scriptPath: path.ParsedPath = path.parse(path.normalize(__dirname + '/../signals.sh'));

/**
 * injectPath represents the `inject` binary that will be searched for in the directory specified by the localDir value
 * of the execaConfig object.
 */
const injectPath: path.ParsedPath = path.parse(path.normalize(__dirname + '/../bin/inject'));

function init() {
    // Verify that `injectPath` points to an actual file.
    if (!fsExists(path.format(injectPath))) {
        throw new Error(`invalid inject path: ${path.format(injectPath)}`);
    }
}

/**
 * Run the configured script (defined by `scriptPath`) and send it
 * @param signal
 * @returns
 */
async function run(command: Command, signal: NodeJS.Signals): Promise<Boolean> {
    // Create the execa child process.
    const subprocess: ExecaChildProcess = execa(
        path.format(injectPath),
        ['--', path.format(command.executable), ...(command.arguments ?? [])],
        execaConfig,
    );

    // After `signalTimeout` send the requested signal to the subprocess. The subprocess should exit, if it does not,
    // then forcibly terminate the subprocess after `killTimeout` passes.
    setTimeout(() => {
        subprocess.kill(signal, {
            forceKillAfterTimeout: command.options.killTimeout,
        });
    }, command.options.signalTimeout);

    // Wait for command to complete or fail and retrieve output.
    const outputLines: string[] = [];
    try {
        const completedSubprocess = (await subprocess) as ExecaReturnValue;
        outputLines.push(...completedSubprocess.stdout.split('\n'));
    } catch (error) {
        if (error instanceof Error && Object.prototype.hasOwnProperty.call(error, 'stdout')) {
            outputLines.push(...(error as ExecaError).stdout.split('\n'));
        } else {
            throw error;
        }
    }

    // Analyze results by parsing returned outputLines.
    let signalReceived = false;
    for (const [i, line] of outputLines.entries()) {
        console.log(`[${i}]: ${line}`);
        if (line.includes(`${signal} received`)) {
            signalReceived = true;
        }
    }

    return signalReceived;
}

const command: Command = {
    executable: scriptPath,
    options: {
        killTimeout: processKillTimeout,
        signalTimeout: signalTimeout,
    },
};

const signals: NodeJS.Signals[] = ['SIGHUP', 'SIGINT', 'SIGTERM'];

(async function () {
    // Verify configuration.
    try {
        init();
    } catch (error) {
        console.log((error as Error).message);
        process.exit(1);
    }

    // Run tests.
    let successCounter = 0;
    for (const signal of signals) {
        console.log(`signal: ${signal}`);
        const signalReceived = await Promise.resolve(run(command, signal));
        if (signalReceived) {
            console.log('ok');
            successCounter++;
        } else {
            console.log('err');
        }
    }

    successCounter === signals.length ? process.exit(0) : process.exit(1);
})();
