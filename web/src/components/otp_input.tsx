import {
  ChangeEvent,
  ClipboardEvent,
  FocusEvent,
  KeyboardEvent,
  useEffect,
  useRef,
  useState,
} from "react";

interface OTPInputProps {
  value: string;
  numInputs: number;
  onChange: (otp: string) => void;
  shouldAutoFocus?: boolean;
}

const OTPInput = ({
  value,
  numInputs,
  onChange,
  shouldAutoFocus = false,
}: OTPInputProps) => {
  const [activeInput, setActiveInput] = useState(0);
  const inputRefs = useRef<Array<HTMLInputElement | null>>([]);

  const getOTPValue = () => (value ? value.toString().split("") : []);

  useEffect(() => {
    inputRefs.current = inputRefs.current.slice(0, numInputs);
  }, [numInputs]);

  useEffect(() => {
    if (shouldAutoFocus) {
      inputRefs.current[0]?.focus();
    }
  }, [shouldAutoFocus]);

  const isInputValueValid = (value: string) => {
    const isTypeValid = !isNaN(Number(value));
    return isTypeValid && value.trim().length === 1;
  };

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { value } = event.target;

    if (isInputValueValid(value)) {
      changeCodeAtFocus(value);
      focusInput(activeInput + 1);
    }
  };

  const handleInputChange = (event: ChangeEvent<HTMLInputElement>) => {
    if (!isInputValueValid(event.target.value)) {
      event.target.value = "";
    }
  };

  const handleFocus =
    (event: FocusEvent<HTMLInputElement>) => (index: number) => {
      setActiveInput(index);
      event.target.select();
    };

  const handleBlur = () => {
    setActiveInput(activeInput - 1);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    const otp = getOTPValue();
    if ([event.code, event.key].includes("Backspace")) {
      event.preventDefault();
      changeCodeAtFocus("");
      focusInput(activeInput - 1);
    } else if (event.code === "Delete") {
      event.preventDefault();
      changeCodeAtFocus("");
    } else if (event.code === "ArrowLeft") {
      event.preventDefault();
      focusInput(activeInput - 1);
    } else if (event.code === "ArrowRight") {
      event.preventDefault();
      focusInput(activeInput + 1);
    } else if (event.key === otp[activeInput]) {
      event.preventDefault();
      focusInput(activeInput + 1);
    } else if (
      event.code === "Spacebar" ||
      event.code === "Space" ||
      event.code === "ArrowUp" ||
      event.code === "ArrowDown"
    ) {
      event.preventDefault();
    }
  };

  const focusInput = (index: number) => {
    const activeInput = Math.max(Math.min(numInputs - 1, index), 0);

    if (inputRefs.current[activeInput]) {
      inputRefs.current[activeInput]?.focus();
      inputRefs.current[activeInput]?.select();
      setActiveInput(activeInput);
    }
  };

  const changeCodeAtFocus = (value: string) => {
    const otp = getOTPValue();
    otp[activeInput] = value[0];
    handleOTPChange(otp);
  };

  const handleOTPChange = (otp: Array<string>) => {
    const otpValue = otp.join("");
    onChange(otpValue);
  };

  const handlePaste = (event: ClipboardEvent<HTMLInputElement>) => {
    event.preventDefault();

    const otp = getOTPValue();
    let nextActiveInput = activeInput;

    const pastedData = event.clipboardData
      .getData("text/plain")
      .slice(0, numInputs - activeInput)
      .split("");

    if (pastedData.some((value) => isNaN(Number(value)))) {
      return;
    }

    for (let pos = 0; pos < numInputs; ++pos) {
      if (pos >= activeInput && pastedData.length > 0) {
        otp[pos] = pastedData.shift() ?? "";
        nextActiveInput++;
      }
    }

    focusInput(nextActiveInput);
    handleOTPChange(otp);
  };

  return (
    <div className="flex flex-row justify-center text-center mt-5" role="group">
      {Array.from({ length: numInputs }, (_, index) => index).map((index) => (
        <input
          key={index}
          className="mx-1 h-12 w-12 text-center text-lg font-medium border-2 border-gray-300 rounded-lg focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-1 bg-white shadow-sm"
          type="text"
          maxLength={1}
          name={"code" + (index + 1)}
          aria-label={"Please enter OTP character " + (index + 1)}
          value={getOTPValue()[index] ?? ""}
          ref={(element) => (inputRefs.current[index] = element)}
          onChange={handleChange}
          onFocus={(event) => handleFocus(event)(index)}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
          autoComplete={"off"}
          inputMode="numeric"
          onInput={handleInputChange}
        />
      ))}
    </div>
  );
};

export type { OTPInputProps };
export default OTPInput;
