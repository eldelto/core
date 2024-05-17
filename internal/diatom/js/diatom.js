// Instructions
const EXIT = 0;
const NOP = 1;
const RET = 2;
const CONST = 3;
const FETCH = 4;
const STORE = 5;
const ADD = 6;
const SUBTRACT = 7;
const MULTIPLY = 8;
const DIVIDE = 9;
const MODULO = 10;
const DUP = 11;
const DROP = 12;
const SWAP = 13;
const OVER = 14;
const CJMP = 15;
const CALL = 16;
const SCALL = 17;
const KEY = 18;
const EMIT = 19;
const EQUALS = 20;
const NOT = 21;
const AND = 22;
const OR = 23;
const LT = 24;
const GT = 25;
const RPOP = 26;
const RPUT = 27;
const RPEEK = 28;
const BFETCH = 29;
const BSTORE = 30;

const wordSize = 4;
const wordMax = 2147483647;
const wordMin = -2147483648;


class Stack {
	#cursor = 0;

	constructor(size) {
		const buffer = new ArrayBuffer(size * wordSize);
		this.data = new Int32Array(buffer);
	}

	push(value) {
		if (this.#cursor + 1 >= this.data.length) {
			throw new Error(`push: stack overflow - cursor: ${this.#cursor}, stack size: ${this.data.length}`);
		}

		this.data[this.#cursor] = value;
		this.#cursor++;
	}


	pop() {
		if (this.#cursor <= 0) {
			throw new Error(`pop: stack underflow - cursor: ${this.#cursor}, stack size: ${this.data.length}`);
		}

		this.#cursor--;
		return this.data[this.#cursor];
	}

	peek() {
		if (this.#cursor <= 0) {
			throw new Error(`peek: stack underflow - cursor: ${this.#cursor}, stack size: ${this.data.length}`);
		}

		return this.data[this.#cursor - 1];
	}
}

function add(a, b) {
	const c = a + b;
	if (c > wordMax) {
		return wordMax
	} else if (c < wordMin) {
		return wordMin
	} else {
		return c
	}
}

function subtract(a, b) {
	const c = a - b;
	if (c > wordMax) {
		return wordMax
	} else if (c < wordMin) {
		return wordMin
	} else {
		return c
	}
}

function multiply(a, b) {
	const c = a * b;
	if (c > wordMax) {
		return wordMax
	} else if (c < wordMin) {
		return wordMin
	} else {
		return c
	}
}
//class Input {
//	// TODO: Implement
//}

const stackSize = 30;
const ioBufferSize = 4096;
const memorySize = 8192;

class DiatomVM {

	#programCounter = 0;
	dataStack = new Stack(stackSize);
	returnStack = new Stack(stackSize);
	//#inputBuffer = new Input();
	//input = some element;
	//output = some element;
	#memory = new Uint8Array(new ArrayBuffer(memorySize));

	validateMemoryAccess(addr) {
		if (addr >= this.#memory.length) {
			throw new Error(`out of bound memory access: programCounter=${this.#programCounter} address=${addr}`);
		}
	}

	fetchByte(addr) {
		this.validateMemoryAccess(addr);
		return this.#memory[addr];
	}

	storeByte(addr, b) {
		this.validateMemoryAccess(addr);
		this.#memory[addr] = b;
	}

	wordToBytes(w) {
		const bytes = new Uint8Array(new ArrayBuffer(wordSize));

		for (let i = 0; i < wordSize; i++) {
			bytes[i] = (w >> (i * 8)) & 0xFF;
		}

		return bytes;
	}

	fetchWord(addr) {
		let w = 0;
		for (let i = 0; i < wordSize; i++) {
			const b = this.fetchByte(addr + i);
			const shift = (wordSize - (i + 1)) * 8;
			w = w | (b << shift);
		}

		return w;
	}

	storeWord(addr, w) {
		const bytes = wordToBytes(w);
		for (let i = 0; i < wordSize; i++) {
			vm.storeByte(addr + (wordSize - (i + 1)), bytes[i])
		}
	}

	load(program) {
		if (program.length > this.#memory.length) {
			throw new Error(`program length (${program.length} bytes) exceeds available memory (${this.#memory.length} bytes)`);
		}
		this.#memory.set(program);
	}

	loadRemote(path) {
		fetch(path).then(response => {
			if (!response.ok) {
				throw new Error(`HTTP error! Status: ${response.status}`);
			}

			const program = new Uint8Array(response.arrayBuffer());
		});
	}

	execute() {
		while (true) {
			const instruction = this.#memory[this.#programCounter];
			console.debug(instruction);

			switch (instruction) {
				case EXIT: {
					console.debug("VM exited normally");
					return;
				}
				case NOP: {
					break;
				}
				case RET: {
					const addr = this.returnStack.pop();
					this.#programCounter = addr;
					continue;
				}
				case CONST: {
					this.#programCounter++;
					const w = this.fetchWord(this.#programCounter);
					this.dataStack.push(w);

					this.#programCounter += wordSize;
					continue
				}
				case FETCH: {
					const addr = this.dataStack.pop();
					const w = this.fetchWord(addr);
					this.dataStack.push(w);
					break;
				}
				case STORE: {
					const addr = this.dataStack.pop();
					const value = vm.dataStack.pop();
					this.storeWord(addr, value);
					break;
				}
				case ADD: {
					const a = this.dataStack.pop();
					const b = this.dataStack.pop();
					this.dataStack.push(add(b, a));
					break;
				}
				case SUBTRACT: {
					const a = this.dataStack.pop();
					const b = this.dataStack.pop();
					this.dataStack.push(subtract(b, a));
					break;
				}
				case MULTIPLY: {
					const a = this.dataStack.pop();
					const b = this.dataStack.pop();
					this.dataStack.push(multiply(b, a));
					break;
				}
				case DIVIDE: {
					const a = this.dataStack.pop();
					const b = this.dataStack.pop();
					this.dataStack.push(b / a);
					break;
				}
				case MODULO: {
					const a = this.dataStack.pop();
					const b = this.dataStack.pop();
					this.dataStack.push(b % a);
					break;
				}
				case DUP: {
					const a = this.dataStack.peek();
					this.dataStack.push(a);
					break;
				}
				case DROP: {
					this.dataStack.pop();
					break;
				}
				default:
					throw new Error(`unknown instruction '${instruction}' at memory address '${this.#programCounter}' - terminating`);
			}

			this.#programCounter++;
		}
	}
	// TODO: Implement
}

/*
	How do I want to use that stuff?

	<script src="diatom.js" />
	<script>
	const vm = DiatomVM.load("my-script.dia");
	vm.execute();
	vm.reset();
	</script>

	Handling Javascript events would immediately require some sort of
	event-loop/async programming capabilities but this is out of scope
	for now.
*/


//const s = new Stack(30);
//s.push(11);
//console.log(s.peek());
//console.log(s.pop());
//console.log(s);

//const program = new Uint8Array(new ArrayBuffer(10));
//program.set([11], 0);
//console.log(program);

//const vm = new DiatomVM();
//vm.load(program);
//vm.execute();
