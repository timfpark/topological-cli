const { Processor } = require('topological');

class WriteLocationsProcessor extends Processor {
    process(message, callback) {
        return callback();
    }
}

module.exports = WriteLocationsProcessor;