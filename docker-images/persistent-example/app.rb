require 'sinatra'
require 'json'

set :bind, '0.0.0.0'

get '/' do
    log_file = ENV['LOG_FILE']

    if File.exist? log_file
        last_line = IO.readlines(log_file)[-1]
        last_number = last_line.strip.to_i
    else
        puts "File doesn't exist, creating..."
        File.write(log_file, "")        
        last_number = 0
    end

    num_lines = File.open(log_file, "r").readlines.size

    if num_lines != last_number
        raise "Log files has the wrong line count!"
    end
    
    current_number = last_number + 1

    open(log_file, 'a') { |f|
        f.puts current_number
    }

    {
        last_number: last_number,
        current_number: current_number,
    }.to_json
end