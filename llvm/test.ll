@hello_str = private unnamed_addr constant [15 x i8] c"Hello, World!\0A\00"
@format_str = private unnamed_addr constant [3 x i8] c"%s\00"

declare i32 @printf(i8* nocapture, ...)

define i32 @main() {
    %zero = getelementptr [15 x i8], [15 x i8]* @hello_str, i32 0, i32 0
    %hello_cast = bitcast i8* %zero to i8*
    
    %zero1 = getelementptr [3 x i8], [3 x i8]* @format_str, i32 0, i32 0
    %format_cast = bitcast i8* %zero1 to i8*

    call i32 (i8*, ...) @printf(i8* %format_cast, i8* %hello_cast)
    ret i32 69
}
