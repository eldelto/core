const wordSize = 4;

class Stack {
	// TODO: Implement based on 32 bit typed arrays

	#cursor = 0;
	#data;

	constructor(size) {
		const buffer = new ArrayBuffer(size * wordSize);
		this.#data = new Int32Array(buffer);
	}

	push(value) {
		if (this.#cursor+1 >= this.#data.length) {
			throw `push: stack overflow - cursor: ${this.#cursor}, stack size: ${this.#data.length}`;
		}

		this.#data[this.#cursor] = value;
		this.#cursor++;
	}


	pop() {
		if (this.#cursor <= 0) {
			throw `pop: stack underflow - cursor: ${this.#cursor}, stack size: ${this.#data.length}`;
		}

		this.#cursor--;
		return this.#data[this.#cursor];
	}

	peek() {
		if (this.#cursor <= 0) {
			throw `peek: stack underflow - cursor: ${this.#cursor}, stack size: ${this.#data.length}`;
		}

		return this.#data[this.#cursor-1];
	}
}

const s = new Stack(30);
s.push(11);
console.log(s.peek());
console.log(s.pop());
console.log(s);
s.pop();

//class Input {
//	// TODO: Implement
//}
//
//class DiatomVM {
//	static const stackSize = 30;
//	static const ioBufferSize = 4096;
//	static const memorySize = 8192;
//
//	programCounter = 0;
//	dataStack = new Stack(stackDepth);
//	returnStack = new Stack(stackDepth);
//	inputBuffer = new Input();
//	//input = some element;
//	//output = some element;
//	memory = new Uint8Array();
//
//	// TODO: Implement
//}
