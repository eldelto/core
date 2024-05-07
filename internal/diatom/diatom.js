class Stack {
	// TODO: Implement based on 32 bit typed arrays
}

class Input {
	// TODO: Implement
}

class DiatomVM {
	static const stackDepth = 30;
	static const ioBufferSize = 4096;
	static const memorySize = 8192;

	programCounter = 0;
	dataStack = new Stack(stackDepth);
	returnStack = new Stack(stackDepth);
	inputBuffer = new Input();
	//input = some element;
	//output = some element;
	memory = new Uint8Array();

	// TODO: Implement
}
