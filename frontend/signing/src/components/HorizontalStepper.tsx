import { CheckCircle } from "lucide-react";

interface Step {
  id: string;
  title: string;
  description?: string;
}

interface HorizontalStepperProps {
  steps: Step[];
  currentStep: number;
}

export function HorizontalStepper({ steps, currentStep }: HorizontalStepperProps) {
  return (
    <div data-testid="stepper-container" className="w-full">
      <div className="flex items-center justify-between">
        {steps.map((step, index) => {
          const isActive = currentStep === index;
          const isCompleted = currentStep > index;

          return (
            <div 
              key={step.id} 
              data-testid={`stepper-step-${index}`}
              className="flex items-center flex-1"
            >
              <div className="flex items-center relative">
                {/* Step indicator */}
                <div
                  data-testid={`stepper-step-indicator-${index}`}
                  className={`relative z-10 flex items-center justify-center w-10 h-10 rounded-full transition-all duration-300 ${
                    isCompleted
                      ? "bg-green-500 text-white"
                      : isActive
                      ? "bg-blue-600 text-white ring-4 ring-blue-100"
                      : "bg-gray-200 text-gray-500"
                  }`}
                >
                  {isCompleted ? (
                    <CheckCircle className="w-5 h-5" />
                  ) : (
                    <span className="text-sm font-semibold">{index + 1}</span>
                  )}
                </div>

                {/* Step label */}
                <div className="ml-3 hidden sm:block">
                  <p
                    data-testid={`stepper-step-label-${index}`}
                    className={`text-sm font-medium transition-colors duration-300 ${
                      isActive
                        ? "text-gray-900"
                        : isCompleted
                        ? "text-green-600"
                        : "text-gray-400"
                    }`}
                  >
                    {step.title}
                  </p>
                </div>
              </div>

              {/* Connector line */}
              {index < steps.length - 1 && (
                <div className="flex-1 ml-3">
                  <div
                    className={`h-0.5 transition-colors duration-300 ${
                      isCompleted ? "bg-green-500" : "bg-gray-200"
                    }`}
                  />
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Mobile step labels */}
      <div 
        data-testid="stepper-mobile-label"
        className="sm:hidden mt-4 text-center"
      >
        <p className="text-sm font-medium text-gray-900">
          {steps[currentStep]?.title}
        </p>
        {steps[currentStep]?.description && (
          <p className="text-xs text-gray-500 mt-1">
            {steps[currentStep].description}
          </p>
        )}
      </div>
    </div>
  );
}