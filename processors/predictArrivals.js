const { Processor } = require('topological');

class PredictArrivals extends Processor {
    process(message, callback) {
        return callback();
    }
}

module.exports = PredictArrivals;