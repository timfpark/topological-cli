const { Processor } = require('topological');

class NotifyArrivals extends Processor {
    process(message, callback) {
        return callback();
    }
}

module.exports = NotifyArrivals;